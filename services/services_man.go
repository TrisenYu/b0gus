package services

import (
	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_databases "b0gus/databases"
	b0gus_misc_utils "b0gus/misc_utils"
	"context"
	"path/filepath"
	"sync"
)

type serviceReadCtrl struct {
	Ctx     context.Context
	Data_ch <-chan any
}

type serviceRunnerType[T b0gus_config.AbsServType] func(
	*serviceReadCtrl, *T,
	*b0gus_databases.RecordDB,
	*sync.WaitGroup, ...any,
)

func tRunner[T b0gus_config.AbsServType](
	ctx context.Context,
	ch <-chan any,
	conf any,
	wait_group *sync.WaitGroup,
	db *b0gus_databases.RecordDB,
	runner serviceRunnerType[T],
	args ...any,
) {
	curr, ok := conf.(*T)
	if !ok {
		return
	}
	var link_gadget = serviceReadCtrl{
		Ctx:     ctx,
		Data_ch: ch,
	}
	/* Add wait group */
	wait_group.Add(1)
	go runner(
		&link_gadget, curr,
		db, wait_group, args,
	)
}

func GenericRunner(
	tag string,
	ctx context.Context,
	ch <-chan any,
	conf any,
	wait_group *sync.WaitGroup,
	db *b0gus_databases.RecordDB,
	args ...any,
) {
	if !b0gus_config.InfoEvalator(tag, conf) {
		return
	}
	switch tag {
	case "SSHconfig":
		tRunner(ctx, ch, conf, wait_group, db, SSHserver, args)
	case "NTPconfig":
		tRunner(ctx, ch, conf, wait_group, db, NTPserver, args)
	case "DNSconfig":
		tRunner(ctx, ch, nil, wait_group, db, DNSserver, args)
	case "TelnetConfig":
		tRunner(ctx, ch, nil, wait_group, db, TelnetServer, args)
	default: // unknown tag
		return
	}
}

func GenericArgs(tag string, serv_conf *b0gus_config.LocalConfig) any {
	switch tag {
	case "SSHconfig":
		pem_path, _ := filepath.Abs(filepath.Join(
			b0gus_config.Config_dir_as_str,
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

func Brancher(
	need_shutdown chan struct{},
	server_conf *b0gus_config.LocalConfig,
	db *b0gus_databases.RecordDB, // *gorm.DB, *redis.Client, *mongo.Client
	wait_group *sync.WaitGroup,
) {
	// TODO: length should be directly caculated from b0gus_config.LocalConfig
	// I would like to use reflection instead
	// will tidy as expected
	root_ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var (
		ch_slots  map[string]chan any = make(map[string]chan any, 5)
		exist_map map[string]bool     = make(map[string]bool, 5)
	)

	// todo: iterate struct and create channel but not explict define it
	res_map := b0gus_misc_utils.TurnStruct2Map(server_conf.ServerConfig)
	for k := range res_map {
		switch k {
		case "PemName":
		case "PemType":
		case "PemLen":
		case "DatabaseConfig":
		default:
			ch_slots[k] = make(chan any, 1)
			exist_map[k] = false
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
			cancel() // cancel as a global shutdown convention
			for ex := range exist_map {
				exist_map[ex] = false
			}
			return
		case curr_config, ok := <-b0gus_config.UpdateFlag:
			if !ok {
				goto stuck
			}

			// help for the only database that has registed in frontend
			go b0gus_config.GlobConfigMaintainer.UpdateConfig()

			// TODO: 0. holy shxt we still have to inspect the curr_config to determine whether we
			// 			should shut down/reload a service or not.
			//		 1. inspect curr_res and consider if ch_slots[ex] will be blocked
			//		 2. should be able to invoke a new service as long as
			// 			the configuration of child is confirmed and accepted?
			for serv_tag, val := range exist_map {

				// check service_ex whether should be killed or updated in this loop
				curr_res, _ := b0gus_misc_utils.GetFieldValueByName(
					curr_config.ServerConfig,
					serv_tag,
				) // interface{} needs explictly unwrapping by enforced type convertion,
				// but we only have tag-string
				decision := b0gus_config.InfoEvalator(serv_tag, curr_res)
				if !val { // does not have instantiated task-request
					if !decision {
						continue
					}
					exist_map[serv_tag] = decision
					GenericRunner(
						serv_tag, root_ctx, ch_slots[serv_tag],
						curr_res, wait_group, db,
						GenericArgs(serv_tag, server_conf),
					)
					continue
				} else {
					if !decision {
						exist_map[serv_tag] = decision
						curr_res = struct{}{}
					}
				}
				// and what if the channel is not empty or there is not such service
				// when we second
				ch_slots[serv_tag] <- curr_res
			}
			goto stuck
		}

	}()

	for k := range ch_slots {
		curr_val, err := b0gus_misc_utils.GetFieldValueByName(server_conf.ServerConfig, k)
		if err != nil {
			continue
		}
		exist_map[k] = true
		GenericRunner(
			k, root_ctx, ch_slots[k],
			curr_val, wait_group, db,
			GenericArgs(k, server_conf),
		)
	}

	// check the port given by service whether is in used
	// if so, b0gus won't running this and instead push such information into log
	// if b0gus_config.CheckSSHconfig(&server_conf.ServerConfig.SSHconfig) {
	// 	pem_path, _ := filepath.Abs(filepath.Join(
	// 		b0gus_config.Config_dir_as_str,
	// 		server_conf.ServerConfig.PemName,
	// 	))

	// 	if host_key := b0gus_crypto_aux.LoadOrCreateSSHpem(
	// 		pem_path, server_conf.ServerConfig.PemType,
	// 		server_conf.ServerConfig.PemLen); host_key != nil {

	// 		name := b0gus_misc_utils.GetTypeNameViaType(server_conf.ServerConfig.SSHconfig)
	// 		exist_map[name] = true
	// 		var link_gadget = serviceReadCtrl{
	// 			Ctx:     root_ctx,
	// 			Data_ch: ch_slots[name],
	// 		}
	// 		/* Add wait group */
	// 		wait_group.Add(1)
	// 		go SSHserver(
	// 			&link_gadget,
	// 			&server_conf.ServerConfig.SSHconfig,
	// 			db, wait_group, host_key,
	// 		)
	// 	}
	// }

	// if b0gus_config.CheckTelnetConfig(nil) {
	// 	/* Add wait group */
	// 	name := b0gus_misc_utils.GetTypeNameViaType(server_conf.ServerConfig.TelnetConfig)
	// 	exist_map[name] = true
	// 	var link_gadget = serviceReadCtrl{
	// 		Ctx:     root_ctx,
	// 		Data_ch: ch_slots[name],
	// 	}
	// 	wait_group.Add(1)
	// 	go TelnetServer(
	// 		&link_gadget,
	// 		&server_conf.ServerConfig.TelnetConfig,
	// 		db, wait_group,
	// 	)
	// }
	// if b0gus_config.CheckNTPconfig(&server_conf.ServerConfig.NTPconfig) {
	// 	/* Add wait group */
	// 	name := b0gus_misc_utils.GetTypeNameViaType(server_conf.ServerConfig.NTPconfig)
	// 	var link_gadget = serviceReadCtrl{
	// 		Ctx:     root_ctx,
	// 		Data_ch: ch_slots[name],
	// 	}
	// 	exist_map[name] = true
	// 	wait_group.Add(1)
	// 	go NTPserver(
	// 		&link_gadget,
	// 		&server_conf.ServerConfig.NTPconfig,
	// 		db, wait_group,
	// 	)
	// }
	// if b0gus_config.CheckDNSconfig(nil) {
	// 	name := b0gus_misc_utils.GetTypeNameViaType(server_conf.ServerConfig.DNSconfig)
	// 	var link_gadget = serviceReadCtrl{
	// 		Ctx:     root_ctx,
	// 		Data_ch: ch_slots[name],
	// 	}
	// 	exist_map[name] = true
	// 	wait_group.Add(1)
	// 	go DNSserver(
	// 		&link_gadget,
	// 		&server_conf.ServerConfig.DNSconfig,
	// 		db, wait_group,
	// 	)
	// }
	wait_group.Wait()

	// still need a coroutine to monitor whether the configuration is updated
	// and send corresponding signal/struct to specific service
	//
}
