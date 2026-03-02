package services

import (
	"context"
	"path/filepath"
	"sync"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_misc_utils "b0gus/misc_utils"
)

// Since the definition of SSHserverConf is seperated from b0gus_config
// SetSpecConfViaTag is thus used as a bizzare generic function
// in the scope of golang programming.
func SetSpecConfViaTag(tag string, conf any) b0gus_config.AbsServType {
	switch tag {
	case "SSHconfig":
		res, ok := conf.(b0gus_config.SSHconfig)
		if !ok || !b0gus_config.GenericConfChecker(conf, b0gus_config.CheckSSHconfig) {
			return nil
		}
		var s SSHserverConf
		res.RegistRunner(s.Run)
		return res
	case "NTPconfig":
		res, ok := conf.(b0gus_config.NTPconfig)
		if !ok || !b0gus_config.GenericConfChecker(conf, b0gus_config.CheckNTPconfig) {
			return nil
		}
		var n NTPserverConf
		res.RegistRunner(n.Run)
		return res
	case "DNSconfig":
		res, ok := conf.(b0gus_config.DNSconfig)
		if !ok || !b0gus_config.GenericConfChecker(conf, b0gus_config.CheckDNSconfig) {
			return nil
		}
		res.RegistRunner(nil)
		return res
	case "TelnetConfig":
		res, ok := conf.(b0gus_config.TelnetConfig)
		if !ok || !b0gus_config.GenericConfChecker(conf, b0gus_config.CheckTelnetConfig) {
			return nil
		}
		res.RegistRunner(nil)
		return res
	default: // unknown tag
		return nil
	}
}

// adjust arguments for different services
func GenericArgs(tag string, serv_conf *b0gus_config.LocalConfig) any {
	switch tag {
	case "SSHconfig":
		pem_path, _ := filepath.Abs(filepath.Join(
			b0gus_config.ConfigDirAsStr,
			serv_conf.ServerConfig.PemName,
		))
		return b0gus_crypto_aux.LoadOrCreateSSHpem(
			pem_path, serv_conf.ServerConfig.PemType,
			serv_conf.ServerConfig.PemLen,
		)
	default:
		return nil
	}

}

func updatationHandler(
	curr_config *b0gus_config.LocalConfig,
	db *b0gus_config.RuntimeDB,
	serv_alive_map map[string]bool,
	root_ctx context.Context,
	ch_slots map[string]chan any,
	wait_group *sync.WaitGroup,
) {
	for serv_tag, val := range serv_alive_map {
		// check service_ex whether should be killed or updated in this loop
		curr_conf, _ := b0gus_misc_utils.GetFieldValueByName(
			curr_config.ServerConfig, serv_tag,
		) // interface{}/any needs explictly unwrapping by enforced type convertion,

		// we only have tag-strings
		decision := SetSpecConfViaTag(serv_tag, curr_conf)
		about_to_run := func() {
			decision.InvokeRunner(
				&b0gus_config.ServicesConcurrencyCtrl{
					Ctx:     root_ctx,
					Data_ch: ch_slots[serv_tag],
				}, db,
				GenericArgs(serv_tag, curr_config),
			)
		}
		if val && decision == nil {
			// apparently without side-effects
			serv_alive_map[serv_tag] = false
			curr_conf = struct{}{}
		} else if !val && decision == nil {
			// does not have instantiated task-request
			continue
		} else if !val && decision != nil {
			// startup
			serv_alive_map[serv_tag] = true
			wait_group.Go(about_to_run)
			continue
		}
		ch_slots[serv_tag] <- curr_conf
	}
}

// brancher as the services' steward of b0gus
func Brancher(
	need_shutdown chan struct{},
	server_conf *b0gus_config.LocalConfig,
	db *b0gus_config.RuntimeDB, // *gorm.DB, *mongo.Client
	wait_group *sync.WaitGroup,
) {
	// TODO: length should be directly caculated from b0gus_config.LocalConfig
	// I would like to use reflection instead
	// It will be tidy as expected
	root_ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var (
		ch_slots       map[string]chan any = make(map[string]chan any, 5)
		serv_alive_map map[string]bool     = make(map[string]bool, 5)
	)

	// TODO: iterate struct and create channel but not explictly define it
	res_map := b0gus_misc_utils.TurnStruct2Map(server_conf.ServerConfig)
	for k := range res_map {
		switch k {
		case "PemName":
		case "PemType":
		case "PemLen":
		case "Language":
		case "DatabaseConfig": // TODO
		default:
			ch_slots[k] = make(chan any, 1)
			serv_alive_map[k] = false
		}
	}
	defer func() {
		for k := range ch_slots {
			close(ch_slots[k])
		}
	}()

	go func() {
	stuck:
		select {
		case <-need_shutdown:
			cancel() /* ctx.cancel() used as a global shutdown convention */
			for ex := range serv_alive_map {
				serv_alive_map[ex] = false
			}
			return
		case curr_config, ok := <-b0gus_config.UpdateFlag:
			if !ok {
				goto stuck
			}
			// help for the only database that has registed in frontend
			go b0gus_config.GlobConfigMaintainer.UpdateConfig(curr_config)

			// TODO: 0. we still have to inspect the curr_config to determine whether we
			// 			should shut down/reload a service or not.
			//		 1. inspect curr_conf and consider if ch_slots[ex] will be blocked
			//		 2. should be able to invoke a new service as long as
			// 			the configuration of child is confirmed and accepted?
			updatationHandler(
				curr_config, db,
				serv_alive_map, root_ctx,
				ch_slots, wait_group,
			)
			goto stuck
		}
	}()

	for k := range ch_slots {
		curr_conf, err := b0gus_misc_utils.GetFieldValueByName(server_conf.ServerConfig, k)
		if err != nil {
			continue
		}
		decision := SetSpecConfViaTag(k, curr_conf)
		if decision == nil {
			continue
		}
		serv_alive_map[k] = true
		wait_group.Go(func() {
			decision.InvokeRunner(
				&b0gus_config.ServicesConcurrencyCtrl{
					Ctx:     root_ctx,
					Data_ch: ch_slots[k],
				}, db,
				GenericArgs(k, server_conf),
			)
		})
	}

	wait_group.Wait()
	// still need a coroutine to monitor whether the configuration is updated
	// and send corresponding signal/struct to specific service
}
