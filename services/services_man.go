package services

/// Last modified at 2026/05/18 星期一 15:48:07
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"context"
	"sync"
	"sync/atomic"

	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/internal/misc_utils"
)

type servRunner func(
	globConf *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
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
		// [NOTE]: temporarily do not require for services updates
		//         due to the engineering complexity.
	}()

	// [TODO]: execute all services at one time?
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
		//
		//     any rpc services will return by itself, then the main process should be waiting for
		//     further command or monitoring any available configuration updates
		//
		//      services downgrade should be implemented as well
		wg.Go(func() {
			decision(
				serverConf, &configs.ServConcurrentCtrl{
					TerminatedCtx: rootCtx,
					ServNetTypeCh: chSlots[k],
				}, genericArgs(k, serverConf),
			)
		})
	}
	wg.Wait()
}

// genericArgs adjusts arguments required for different services
// For SSH, default will set LocalConfigPathStr/TLSKeyPath as the in parameter
// Others will provide a nil as the placeholder
func genericArgs(
	tag configs.ServEnum,
	servConf *atomic.Pointer[configs.LocalConfig],
) any {
	if tag != configs.SSHEnum {
		return nil
	}
	if servConf == nil {
		return nil
	}
	snapshot, ok := servConf.Load().SelectTerm(tag).(configs.SSHconfig)
	if !ok {
		return nil
	}
	pemPath := snapshot.TLSKeyPath
	if misc_utils.IsFilePath(pemPath) {
		return crypto_aux.LoadOrCreateSSHpem(pemPath, snapshot.PemType, snapshot.PemLen)
	}
	pemPath = configs.GetFilePathUnderConfigDir(snapshot.PemName)
	if misc_utils.IsFilePath(pemPath) {
		return crypto_aux.LoadOrCreateSSHpem(pemPath, snapshot.PemType, snapshot.PemLen)
	}
	return nil
}
