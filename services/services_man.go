package services

/// Last modified at 2026/05/09 星期六 15:18:24
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"

	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/databases"
)

type servRunner func(
	globConf *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
	db databases.DBhandler,
	args ...any,
)

var (
	tagMapsToServ = map[configs.ServEnum]servRunner{
		configs.SSHEnum:  (&SSHServConf{}).Run,
		configs.NTPEnum:  (&NTPServConf{}).Run,
		configs.DNSEnum:  (&DNSservConf{}).Run,
		configs.SMTPEnum: (&SMTPServConf{}).Run,
		configs.HTTPEnum: (&HTTPservConf{}).Run,
	}
)

// Brancher distributes and initiates every available services of b0gus
func Brancher(
	terminator <-chan struct{},
	serverConf *atomic.Pointer[configs.LocalConfig],
	db *databases.RuntimeDB,
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
		// [FEAT]: temporarily do not require for services updates
		//         due to the engineering complexity .
	}()

	for k := range chSlots {
		decision, ok := tagMapsToServ[k]
		if !ok || decision == nil {
			continue
		}
		servAliveMap[k] = true
		// [TODO]: implement probe mechanism for RPC.
		//      it can mitigate the intensive pressure on current host
		// 		when the computing capacity of host is not adequate and robust.
		// 		Any configuration upon RPC has to be inspected here.
		wg.Go(func() {
			decision(
				serverConf, &configs.ServConcurrentCtrl{
					Ctx:           rootCtx,
					ServNetTypeCh: chSlots[k],
				}, db, genericArgs(k, serverConf),
			)
		})
	}
	wg.Wait()
}

// genericArgs adjusts arguments required for different services
func genericArgs(
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
