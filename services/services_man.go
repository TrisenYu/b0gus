package services

import (
	"context"
	"path/filepath"
	"sync"

	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/misc_utils"
)

// SetSpecConfViaTag is thus used as a bizarre generic function
// in the scope of golang programming, due to the definition of SSHServConf
// is separated from configs
func SetSpecConfViaTag(tag string, conf any) configs.AbsServType {
	switch tag {
	case "SSHconfig":
		res, ok := conf.(configs.SSHconfig)
		if !ok || !configs.GenericConfChecker(conf, configs.CheckSSHconfig) {
			return nil
		}
		var s SSHServConf
		res.RegisterRunner(s.Run)
		return res
	case "NTPconfig":
		res, ok := conf.(configs.NTPconfig)
		if !ok || !configs.GenericConfChecker(conf, configs.CheckNTPconfig) {
			return nil
		}
		var n NTPServConf
		res.RegisterRunner(n.Run)
		return res
	case "DNSconfig":
		res, ok := conf.(configs.DNSconfig)
		if !ok || !configs.GenericConfChecker(conf, configs.CheckDNSconfig) {
			return nil
		}
		res.RegisterRunner(nil)
		return res
	case "SMTPconfig":
		res, ok := conf.(configs.SMTPconfig)
		if !ok || !configs.GenericConfChecker(conf, configs.CheckSMTPconfig) { // TODO
			return nil
		}
		var s SMTPServConf
		res.RegisterRunner(s.Run)
		return res
	default: // unknown tag
		return nil
	}
}

// GenericArgs adjusts arguments for different services
func GenericArgs(tag string, servConf *configs.LocalConfig) any {
	switch tag {
	case "SSHconfig":
		pemPath, _ := filepath.Abs(filepath.Join(
			filepath.Dir(configs.LocalConfigPathAsStr),
			servConf.ServerConfig.PemName,
		))
		return crypto_aux.LoadOrCreateSSHpem(
			pemPath, servConf.ServerConfig.PemType,
			servConf.ServerConfig.PemLen,
		)
	default:
		return nil
	}
}

// Brancher manages every available services of b0gus
func Brancher(
	needShutdown <-chan struct{},
	serverConf *configs.LocalConfig,
	db *configs.RuntimeDB, /* *gorm.DB, *mongo.Client */
) {
	var (
		wg           sync.WaitGroup
		chSlots      = make(map[string]chan any)
		servAliveMap = make(map[string]bool)
	)
	rootCtx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		for k := range chSlots {
			close(chSlots[k])
		}
	}()

	// TODO: iterate struct and create channel but not explicitly define it
	resMap := misc_utils.TurnStruct2Map(serverConf.ServerConfig)
	for k := range resMap {
		switch k {
		case "PemName":
		case "PemType":
		case "PemLen":
		case "Language":
		case "RecDBConfig":
		default:
			chSlots[k] = make(chan any, 1)
			servAliveMap[k] = false
		}
	}

	go func() {
	stuck:
		select {
		case <-needShutdown:
			cancel() /* ctx.cancel() used as a global shutdown convention */
			for ex := range servAliveMap {
				servAliveMap[ex] = false
			}
			return
		case currConfig, ok := <-configs.UpdateFlag:
			if !ok {
				goto stuck
			}
			// help for the only database that has registered in frontend
			// go configs.GlobConfigMan.UpdateConfig(currConfig)

			for servTag, val := range servAliveMap {
				// check service_ex whether it should be killed or updated in this loop
				currConf, _ := misc_utils.GetFieldValueByName(
					currConfig.ServerConfig, servTag,
				) // interface{}/any needs explicitly unwrapping by enforced type convertion,

				// we only have tag-strings
				decision := SetSpecConfViaTag(servTag, currConf)
				aboutToRun := func() {
					decision.InvokeRunner(
						&configs.ServConcurrentCtrl{
							Ctx:    rootCtx,
							DataCh: chSlots[servTag],
						}, db,
						GenericArgs(servTag, currConfig),
					)
				}
				if val && decision == nil {
					// apparently without side effects
					servAliveMap[servTag] = false
					currConf = struct{}{}
				} else if !val && decision == nil {
					// does not have instantiated task-request
					continue
				} else if !val && decision != nil {
					// startup
					servAliveMap[servTag] = true
					wg.Go(aboutToRun)
					continue
				}
				chSlots[servTag] <- currConf
			}
			goto stuck
		}
	}()

	for k := range chSlots {
		currConf, err := misc_utils.GetFieldValueByName(serverConf.ServerConfig, k)
		if err != nil {
			continue
		}
		decision := SetSpecConfViaTag(k, currConf)
		if decision == nil {
			continue
		}
		servAliveMap[k] = true
		wg.Go(func() {
			decision.InvokeRunner(
				&configs.ServConcurrentCtrl{
					Ctx:    rootCtx,
					DataCh: chSlots[k],
				}, db,
				GenericArgs(k, serverConf),
			)
		})
	}
	wg.Wait()
}
