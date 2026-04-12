package services

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"

	"b0gus/configs"
	"b0gus/crypto_aux"
)

type servRunner func(
	ConfObj *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
	db *configs.RuntimeDB,
	args ...any,
)

// Brancher manages every available services of b0gus-arm64
func Brancher(
	terminator <-chan struct{},
	serverConf *atomic.Pointer[configs.LocalConfig],
	db *configs.RuntimeDB,
) {
	var (
		wg           sync.WaitGroup
		chSlots      = make(map[configs.ServEnum]chan string)
		servAliveMap = make(map[configs.ServEnum]bool)
	)
	rootCtx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		for k := range chSlots {
			close(chSlots[k])
		}
	}()

	for i := configs.RawEnum + 1; i < configs.ENDofEnum; i++ {
		chSlots[i] = make(chan string)
	}
	go func() {
		<-terminator
		cancel()
		for ex := range servAliveMap {
			servAliveMap[ex] = false
		}
		// [FEAT]: temporary do not require for services updates
		//         due to the engineering complexity .
	}()

	for k := range chSlots {
		decision := setSpecConfViaTag(k)
		if decision == nil {
			continue
		}
		servAliveMap[k] = true
		// [TODO]: RPC. it can mitigate the intensive pressure on current host
		// 		when its computing capacity is not robust.
		// 		any configuration upon RPC has to be inspected here.
		wg.Go(func() {
			decision(
				serverConf, &configs.ServConcurrentCtrl{
					Ctx:           rootCtx,
					ServNetTypeCh: chSlots[k],
				}, db, GenericArgs(k, serverConf),
			)
		})
	}
	wg.Wait()
}

// setSpecConfViaTag is thus used as a bizarre generic function
// in the scope of golang programming, due to the definition of SSHServConf
// is separated from configs
func setSpecConfViaTag(tag configs.ServEnum) servRunner {
	switch tag {
	case configs.SSHEnum:
		return (&SSHServConf{}).Run
	case configs.NTPEnum:
		return (&NTPServConf{}).Run
	case configs.DNSEnum:
		return (&DNSserverConf{}).Run
	case configs.SMTPEnum:
		return (&SMTPServConf{}).Run
	case configs.HTTPEnum:
		return (&HTTPservConf{}).Run
	default: // unknown tag
		return nil
	}
}

// GenericArgs adjusts arguments for different services
func GenericArgs(
	tag configs.ServEnum,
	servConf *atomic.Pointer[configs.LocalConfig],
) any {
	switch tag {
	case configs.SSHEnum:
		snapshot, ok := servConf.Load().SelectTerm(configs.SSHEnum).(configs.SSHconfig)
		if !ok {
			return nil
		}
		pemPath, _ := filepath.Abs(filepath.Join(
			filepath.Dir(configs.LocalConfigPathAsStr),
			snapshot.PemName,
		))
		return crypto_aux.LoadOrCreateSSHpem(pemPath, snapshot.PemType, snapshot.PemLen)
	default:
		return nil
	}
}
