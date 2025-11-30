package services

import (
	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_databases "b0gus/databases"
	b0gus_datatypes "b0gus/generic_datatypes"
	"path/filepath"
	"sync"
)

func Brancher(
	need_shutdown *b0gus_datatypes.ConcurrentCtrl,
	server_conf *b0gus_config.LocalConfig,
	db *b0gus_databases.RecordDB, // *gorm.DB, *redis.Client, *mongo.Client
	wait_group *sync.WaitGroup,
) {
	// check the port given by service whether is in used
	// if so, b0gus won't running this and instead push such information into log
	if b0gus_config.CheckSSHconfig(&server_conf.ServerConfig.SSH) {
		pem_path, _ := filepath.Abs(filepath.Join(
			b0gus_config.Config_dir_as_str,
			server_conf.ServerConfig.PemName,
		))
		if host_key := b0gus_crypto_aux.LoadOrCreateSSHpem(
			pem_path, server_conf.ServerConfig.PemType,
			server_conf.ServerConfig.PemLen); host_key != nil {
			/* Add wait group */
			wait_group.Add(1)
			go SSHserver(
				need_shutdown,
				&server_conf.ServerConfig.SSH,
				host_key, db, wait_group,
			)
		}
	}
	if b0gus_config.CheckTelnetConfig(nil) {
		/* Add wait group */
		wait_group.Add(1)
		go TelnetServer(
			need_shutdown,
			&server_conf.ServerConfig.Telnet,
			db, wait_group,
		)
	}
	if b0gus_config.CheckNTPconfig(&server_conf.ServerConfig.NTP) {
		/* Add wait group */
		wait_group.Add(1)
		go NTPserver(
			need_shutdown,
			&server_conf.ServerConfig.NTP,
			db, wait_group,
		)
	}
	if b0gus_config.CheckDNSconfig(nil) {
		wait_group.Add(1)
		go DNSserver(
			need_shutdown,
			&server_conf.ServerConfig.DNS,
			db, wait_group,
		)
	}
	wait_group.Wait()
}
