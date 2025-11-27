// SPDX-LICENSE-IDENTIFIER: 3-Clause-BSD
package main

import (
	"os"
	"os/signal"
	"path/filepath"

	"sync"
	"syscall"

	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_datatypes "b0gus/datatypes"
	b0gus_services "b0gus/services"
)

func servicesBrancher(
	need_shutdown *b0gus_datatypes.ConcurrentCtrl,
	conf_path string,
	server_conf *b0gus_config.LocalConfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	// check the port given by service whether is in used
	// if so, b0gus won't running this and instead push such information into log
	if b0gus_config.CheckSSHconfig(&server_conf.ServerConfig.SSH) {
		pem_path, _ := filepath.Abs(filepath.Join(conf_path, server_conf.ServerConfig.PemName))
		if host_key := b0gus_crypto_aux.LoadOrCreateSSHpem(
			pem_path, server_conf.ServerConfig.PemType,
			server_conf.ServerConfig.PemLen); host_key != nil {
			/* Add wait group */
			wait_group.Add(1)
			go b0gus_services.SSHserver(
				need_shutdown, &server_conf.ServerConfig.SSH,
				host_key, db, wait_group,
			)
		}
	}
	if b0gus_config.CheckTelnetConfig(nil) {
		/* Add wait group */
		wait_group.Add(1)
		go b0gus_services.TelnetServer(
			need_shutdown, &server_conf.ServerConfig.Telnet,
			db, wait_group,
		)
	}
	if b0gus_config.CheckNTPconfig(&server_conf.ServerConfig.NTP) {
		/* Add wait group */
		wait_group.Add(1)
		go b0gus_services.NTPserver(
			need_shutdown, &server_conf.ServerConfig.NTP,
			db, wait_group,
		)
	}
	wait_group.Wait()
}

func B0gusRun() {
	// TODO: neccessary executable binary files/dependencies imediate check inside b0gus
	conf_dir_str, _ := filepath.Abs(b0gus_config.Config_dir_as_str)
	bogus_conf := b0gus_config.LoadDefaultConfig("")
	db, db_str, err := b0gus_config.SelectDatabaseBackend(bogus_conf.ServerConfig.Database)
	// **Connect** to Database. Create table when ensuring to run the server
	// TODO: should we 'hot-plug' the configurations of database and apply them with concurrent protections?
	if err != nil {
		b0gus_config.Logger.Errorf(
			"Failed to connect to %s due to %s",
			db_str, err.Error(),
		)
		return
	} else if db == nil {
		b0gus_config.Logger.Error("Empty database file descriptor")
		return
	}
	var db_update_callback = func() {
		tmp_db, tmp_db_str, _err := b0gus_config.SelectDatabaseBackend(bogus_conf.ServerConfig.Database)
		if tmp_db == nil || _err != nil || tmp_db_str == "" {
			b0gus_config.Logger.Warn("Won't update the database handler")
			return
		}
		/* Guard the database with concurrent protections */
		b0gus_config.Logger.Error(
			"Won't update the database at present due to the complexity in different transactions of services",
		)
	}

	/* signal notification to terminate the whole server gracefully */
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	var (
		wait_group sync.WaitGroup
		terminator *b0gus_datatypes.ConcurrentCtrl = &b0gus_datatypes.ConcurrentCtrl{
			Ch: make(chan struct{}, 1),
		}
	)
	terminator.Flag.Store(false)

	b0gus_config.GlobConfigMaintainer.Init()
	go b0gus_config.GlobConfigMaintainer.Regist(db_update_callback)
	go func() {
	next_select:
		select {
		case sig := <-signalChan:
			b0gus_config.Logger.Warnf(
				"Catch an OS signal<%s> for terminating b0gus server",
				sig.String(),
			)
			terminator.Flag.Store(true)
			terminator.Ch <- struct{}{}
			close(terminator.Ch)
			close(signalChan)
			signal.Stop(signalChan)

		case _, ok := <-b0gus_config.UpdateFlag:
			if ok {
				b0gus_config.GlobConfigMaintainer.UpdateConfig()
			}
		}

		if !terminator.Flag.Load() {
			goto next_select
		}
	}()
	servicesBrancher(
		terminator, conf_dir_str,
		bogus_conf, db, &wait_group,
	)
	b0gus_config.GlobConfigMaintainer.SelfDestroy()
	terminator.Flag.Store(true)
	close(b0gus_config.UpdateFlag)
}

/* The entry of b0gus. configuration in `./configs/` should be properly set up before executing */
func main() {
	B0gusRun()
}
