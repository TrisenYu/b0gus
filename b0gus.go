// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package main

import (
	"os"
	"os/signal"

	"sync"
	"syscall"

	b0gus_assets "b0gus/assets"
	b0gus_config "b0gus/configs"
	b0gus_services "b0gus/services"
)

/* Overview:

          +-> record_database
	conf -+                     +--> ssh 	(almost there)
	      |         b0gus       |--> telnet (draft)
		  +-> services_manager -+--> smtp 	(draft)
		                        |--> ftp 	(draft)
								|--> ntp 	(almost there)
								|--> database services (not even a draft)
								+--> dns 	(draft)
								...
	configuration-directed honeypot
*/

// The entry of b0gus. configuration in `./configs/` should be properly set up before executing
func main() {
	bogusConf := b0gus_config.LoadDefaultConfig("")
	var globRecordDb = b0gus_config.RuntimeDB{}

	db, dbStr, err := b0gus_config.SelectDatabaseBackend(
		&bogusConf.ServerConfig.RecDBConfig,
	)
	// **Connect** to Database. Create table when being ok to run the server
	if err != nil {
		payload := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"main.DatabaseConnectionError",
			map[string]any{
				"DatabaseStr": dbStr,
				"ErrInfo":     err.Error(),
			},
		)
		b0gus_config.Logger.Error(payload)
		return
	} else if db == nil {
		payload := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"main.DatabaseEmptyError", nil,
		)
		b0gus_config.Logger.Error(payload)
		return
	}

	globRecordDb.AlterDatabaseHandler(db)

	dbUpdateCallback := func(updConfig *b0gus_config.LocalConfig) {
		tmpDB, tmpDBstr, _err := b0gus_config.SelectDatabaseBackend(
			&updConfig.ServerConfig.RecDBConfig,
		)
		if tmpDB == nil || _err != nil || tmpDBstr == "" {
			payload := b0gus_assets.GetLocalizedMsg(
				b0gus_config.GetLang(),
				"main.DatabaseChangingWarn", nil,
			)
			b0gus_config.Logger.Warn(payload)
			return
		}
		globRecordDb.AlterDatabaseHandler(tmpDB)
	}
	var (
		waitGroup  sync.WaitGroup
		terminator = make(chan struct{}, 1)
		signalChan = make(chan os.Signal, 1)
	)

	// signal notification to terminate the whole server gracefully
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	b0gus_config.GlobConfigMaintainer.Init()
	go b0gus_config.GlobConfigMaintainer.Regist(dbUpdateCallback)
	go func() {
		sig := <-signalChan
		payload := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"main.TerminationSignalWarn",
			map[string]any{"SignalStr": sig.String()},
		)
		b0gus_config.Logger.Warn(payload)
		terminator <- struct{}{}
		close(terminator)
		signal.Stop(signalChan)
		close(signalChan)
	}()
	b0gus_services.Brancher(
		terminator, bogusConf,
		&globRecordDb, &waitGroup,
	)
	b0gus_config.GlobConfigMaintainer.SelfDestroy()
	close(b0gus_config.UpdateFlag)
}
