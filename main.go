// Last modified at 2025/11/15 星期六 22:22:42
package main

import (
	"net"
	"os"
	"os/signal"
	"path/filepath"

	"sync"
	"syscall"
	"time"

	ssh "golang.org/x/crypto/ssh"
	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_datatypes "b0gus/datatypes"
	b0gus_misc_utils "b0gus/misc_utils"
	b0gus_services "b0gus/services"
)

func b0gusSSHserver(
	need_shutdown *b0gus_datatypes.ConcurrentCtrl,
	ssh_conf_obj *b0gus_config.SSHconfig,
	host_key ssh.Signer,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	defer wait_group.Done()

	// ssh-related tables
	err := db.AutoMigrate(
		&b0gus_datatypes.AddrInfoDef{},
		&b0gus_datatypes.PortInfoDef{},
		&b0gus_datatypes.UsernameDef{},
		&b0gus_datatypes.SSHClientversionStrDef{},
		&b0gus_datatypes.PubInfoDef{},
		&b0gus_datatypes.PassInfoDef{},
		&b0gus_datatypes.CommandTextDef{},
		// Relation Tables
		&b0gus_datatypes.PortNameRelated{},
		&b0gus_datatypes.PortVerRelated{},
		&b0gus_datatypes.PortPubKeyRelated{},
		&b0gus_datatypes.PortPassRelated{},
		&b0gus_datatypes.PortCmdRelated{},
	)
	if err != nil {
		b0gus_config.Logger.Error(err)
		return
	}
	// Fill SSH Server Configuration with definitions in config file
	ssh_server_conf := b0gus_services.SSHserverConf{
		Addr:              ssh_conf_obj.ListenAddr,
		Port:              ssh_conf_obj.ListenPort,
		MaxClientNum:      ssh_conf_obj.MaxClientNum,
		ClientConnTimeout: time.Duration(ssh_conf_obj.ClientConnTimeout) * time.Second,
		DB_fd:             db,
		PermitLogin:       ssh_conf_obj.PermitLogin,
		EmptyShell:        ssh_conf_obj.EmptyShell,
	}
	ssh_server_conf.TCPListenerSwitchDone = make(chan struct{})
	defer close(ssh_server_conf.TCPListenerSwitchDone)
	ssh_server_conf.ConfigGenericCtrl.Ch = make(chan struct{})
	defer close(ssh_server_conf.ConfigGenericCtrl.Ch)
	ssh_server_conf.ConfigGenericCtrl.Flag.Store(false)
	// callback function for updating when there is any the modification in the monitored configuration file
	var ssh_callback = func() {
		b0gus_config.Logger.Infof("Renew SSH configuration: %v", ssh_conf_obj)
		ssh_server_conf.UpdateConfig(ssh_conf_obj)
	}
	go b0gus_config.GlobConfigMaintainer.Regist(ssh_callback)
	ssh_server_conf.SSHMaliciousClientHandler(need_shutdown, host_key)
}

func b0gusTelnetServer(
	need_shutdown *b0gus_datatypes.ConcurrentCtrl,
	bogus_conf *b0gus_config.TelnetConfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	wait_group.Done()
}

func checkTelnetConfig(telnet_conf *b0gus_config.TelnetConfig) bool {
	return telnet_conf != nil &&
		net.ParseIP(telnet_conf.ListenAddr) != nil &&
		telnet_conf.ListenPort > 1024
}

func checkNTPconfig(ntp_conf *b0gus_config.NTPconfig) bool {
	return false
}

func servicesBrancher(
	need_shutdown *b0gus_datatypes.ConcurrentCtrl,
	conf_path string,
	server_conf *b0gus_config.LocalConfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	// check the port given by service whether is in used
	// if so, b0gus won't running this and instead push such information into log
	conf_mapper := b0gus_misc_utils.TurnStruct2Map(*server_conf)
	if conf_mapper == nil {
		b0gus_config.Logger.Fatal("Unable to reflect config to `map[string]any`!")
	}
	server_conf_mapper := conf_mapper["ServerConfig"]

	// so that we can run go routine in this `for loop`
	// Even though we can directly check SSH/TelnetConfig in ServerConfig struct
	for k := range server_conf_mapper.(map[string]any) {
		switch k {
		case "PemName":
			fallthrough
		case "Database":
			continue // do nothing
		case "SSH":
			if !b0gus_config.CheckSSHconfig(&server_conf.ServerConfig.SSH) {
				continue
			}
			// run given service in different go routine if the necessary fields are not empty
			var host_key ssh.Signer
			pem_path, _ := filepath.Abs(filepath.Join(conf_path, server_conf.ServerConfig.PemName))
			pem_obj, err := b0gus_crypto_aux.LoadHostPem(pem_path)
			if err != nil {
				b0gus_config.Logger.WithField("Err", err).Warn(
					`
Pem seems to be invalid or unsupported, b0gus will create one and store it to the path you assigned`,
				)
				host_key, err = b0gus_crypto_aux.CreatePem(pem_path)
				if err != nil {
					b0gus_config.Logger.Errorf(
						"Unable to create pem at %s due to %s",
						pem_path,
						err.Error(),
					)
					continue
				}
			} else {
				host_key = pem_obj
			}
			wait_group.Add(1)
			go b0gusSSHserver(
				need_shutdown, &server_conf.ServerConfig.SSH,
				host_key, db, wait_group,
			)
			// Understanding how `wait_group.Go()` work
		case "Telnet":
			b0gus_config.Logger.Warn("Yet to implement b0gus telnet shell!")
			if !checkTelnetConfig(nil) {
				continue
			}
			wait_group.Add(1)
			go b0gusTelnetServer(
				need_shutdown, &server_conf.ServerConfig.Telnet,
				db, wait_group,
			)
		case "NTP":
			if !checkNTPconfig(nil) {
				continue
			}
		default:
			b0gus_config.Logger.Errorf("Unknown field<%s> was found in configuration", k)
			continue
		}
	}
	wait_group.Wait()
}

func B0gusRun() {
	// TODO: neccessary executable binary files/dependencies imediate check inside b0gus
	conf_dir_str, _ := filepath.Abs(b0gus_config.Config_dir_as_str)
	bogus_conf := b0gus_config.LoadDefaultConfig("")
	db, db_str, err := b0gus_config.SelectDatabaseBackend(bogus_conf.ServerConfig.Database)
	// **Connect** to Database. Create table when ensuring to run the server
	// TODO: should we hot-plug the configurations of database and apply them with concurrent protections?
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
			b0gus_config.Logger.Warn("won't update the database handler")
			return
		}
		// TODO: Guard the database with concurrent protections
		b0gus_config.Logger.Error(
			"We won't update the database at present due to the complexity in different transactions of services",
		)
	}

	/* signal notification to terminate the ssh server gracefully */
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	var (
		wait_group sync.WaitGroup
		terminator b0gus_datatypes.ConcurrentCtrl
	)
	terminator.Ch = make(chan struct{})
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
		case <-b0gus_config.UpdateFlag:
			b0gus_config.GlobConfigMaintainer.UpdateConfig()
		}

		if !terminator.Flag.Load() {
			goto next_select
		}
	}()
	servicesBrancher(
		&terminator,
		conf_dir_str,
		bogus_conf,
		db, &wait_group,
	)
	b0gus_config.GlobConfigMaintainer.SelfDestroy()
	terminator.Flag.Store(true)
	close(b0gus_config.UpdateFlag)
}

/*  The entry of b0gus
 * configuration in `./configs/` should be set up before executing
 */
func main() { B0gusRun() }
