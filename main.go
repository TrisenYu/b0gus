// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package main

import (
	"os"
	"os/signal"

	"sync"
	"syscall"

	b0gus_config "b0gus/configs"
	bogus_databases "b0gus/databases"
	b0gus_services "b0gus/services"
)

func B0gusRun() {
	// TODO: neccessary executable binary files/dependencies imediate check inside b0gus
	bogus_conf := b0gus_config.LoadDefaultConfig("")

	var glob_record_db = bogus_databases.RecordDB{}

	db, db_str, err := b0gus_config.SelectDatabaseBackend(
		bogus_conf.ServerConfig.DatabaseConfig,
	)
	// **Connect** to Database. Create table when ensuring to run the server
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

	glob_record_db.AlterDatabaseHandler(db)

	var db_update_callback = func() {
		tmp_db, tmp_db_str, _err := b0gus_config.SelectDatabaseBackend(
			bogus_conf.ServerConfig.DatabaseConfig,
		)
		if tmp_db == nil || _err != nil || tmp_db_str == "" {
			b0gus_config.Logger.Warn("Won't update the database handler")
			return
		}
		glob_record_db.AlterDatabaseHandler(tmp_db)
	}

	// TODO: move function below to certain package
	/* signal notification to terminate the whole server gracefully */
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	var (
		wait_group sync.WaitGroup
		terminator = make(chan struct{}, 1)
	)

	b0gus_config.GlobConfigMaintainer.Init()
	go b0gus_config.GlobConfigMaintainer.Regist(db_update_callback)
	go func() {
		sig := <-signalChan
		b0gus_config.Logger.Warnf(
			"Catch an OS signal<%s> for terminating b0gus server",
			sig.String(),
		)
		terminator <- struct{}{}
		close(terminator)
		signal.Stop(signalChan)
		close(signalChan)
	}()
	b0gus_services.Brancher(terminator, bogus_conf, &glob_record_db, &wait_group)
	b0gus_config.GlobConfigMaintainer.SelfDestroy()
	close(b0gus_config.UpdateFlag)
}

/* The entry of b0gus. configuration in `./configs/` should be properly set up before executing */
func main() {
	B0gusRun()
}
