package services

import (
	"context"
	"path/filepath"
	"sync"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_databases "b0gus/databases"
	b0gus_misc_utils "b0gus/misc_utils"
)

type serviceReadCtrl struct {
	Ctx     context.Context
	Data_ch <-chan any
}

type serviceRunnerType[T b0gus_config.AbsServType] func(
	*serviceReadCtrl, *T,
	*b0gus_databases.RuntimeDB,
	...any,
)

func tRunner[T b0gus_config.AbsServType](
	ctx context.Context,
	ch <-chan any,
	conf any,
	wait_group *sync.WaitGroup,
	db *b0gus_databases.RuntimeDB,
	runner serviceRunnerType[T],
	runner_args ...any,
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
	runner(
		&link_gadget, curr,
		db, runner_args,
	)
	wait_group.Done()
}

func GenericRunner(
	tag string,
	ctx context.Context,
	ch <-chan any,
	conf any,
	wait_group *sync.WaitGroup,
	db *b0gus_databases.RuntimeDB,
	runner_extra_args ...any,
) {
	if !b0gus_config.InfoEvalator(tag, conf) {
		return
	}
	switch tag {
	case "SSHconfig":
		tRunner(ctx, ch, conf, wait_group, db, SSHserver, runner_extra_args)
	case "NTPconfig":
		tRunner(ctx, ch, conf, wait_group, db, NTPserver, runner_extra_args)
	case "DNSconfig":
		tRunner(ctx, ch, nil, wait_group, db, DNSserver, runner_extra_args)
	case "TelnetConfig":
		tRunner(ctx, ch, nil, wait_group, db, TelnetServer, runner_extra_args)
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

// brancher as the services' steward of b0gus
func Brancher(
	need_shutdown chan struct{},
	server_conf *b0gus_config.LocalConfig,
	db *b0gus_databases.RuntimeDB, // *gorm.DB, *mongo.Client
	wait_group *sync.WaitGroup,
) {
	// TODO: length should be directly caculated from b0gus_config.LocalConfig
	// I would like to use reflection instead
	// It will be tidy as expected
	root_ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var (
		ch_slots  map[string]chan any = make(map[string]chan any, 5)
		exist_map map[string]bool     = make(map[string]bool, 5)
	)

	// TODO: iterate struct and create channel but not explictly define it
	res_map := b0gus_misc_utils.TurnStruct2Map(server_conf.ServerConfig)
	for k := range res_map {
		switch k {
		case "PemName":
		case "PemType":
		case "PemLen":
		case "DatabaseConfig":
		case "Language":
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
			cancel() /* ctx.cancel() used as a global shutdown convention */
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

			// TODO: 0. we still have to inspect the curr_config to determine whether we
			// 			should shut down/reload a service or not.
			//		 1. inspect curr_res and consider if ch_slots[ex] will be blocked
			//		 2. should be able to invoke a new service as long as
			// 			the configuration of child is confirmed and accepted?
			for serv_tag, val := range exist_map {

				// check service_ex whether should be killed or updated in this loop
				curr_res, _ := b0gus_misc_utils.GetFieldValueByName(
					curr_config.ServerConfig, serv_tag,
				) // interface{} needs explictly unwrapping by enforced type convertion,

				// but we only have tag-string
				decision := b0gus_config.InfoEvalator(serv_tag, curr_res)
				if val && !decision {
					exist_map[serv_tag] = decision
					curr_res = struct{}{}
				} else if !val && !decision { // does not have instantiated task-request
					continue
				} else if !val && decision { // startup
					exist_map[serv_tag] = decision
					go GenericRunner(
						serv_tag, root_ctx, ch_slots[serv_tag],
						curr_res, wait_group, db,
						GenericArgs(serv_tag, server_conf),
					)
					continue
				}
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
		go GenericRunner(
			k, root_ctx, ch_slots[k],
			curr_val, wait_group, db,
			GenericArgs(k, server_conf),
		)
	}

	wait_group.Wait()

	// still need a coroutine to monitor whether the configuration is updated
	// and send corresponding signal/struct to specific service
}
