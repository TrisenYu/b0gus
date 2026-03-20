package main

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"b0gus/configs"
	"b0gus/services"
)

var (
	bogusConf    *configs.LocalConfig
	versionStr   string
	buildTimeStr string
)

func initConfFlags() {
	flag.StringVar(
		&configs.LocalConfigPathAsStr, "conf-path", "./configs/config.toml",
		configs.GetLocalizedMsg("meta_conf.ConfPath", nil),
	)
	bogusConf = configs.LoadDefaultConfig("")

	f := flag.Lookup("conf-path")
	if f != nil {
		// reset the configuration path description if feasible
		f.Usage = configs.GetLocalizedMsg("meta_conf.ConfPath", nil)
	}
	showVersion := flag.Bool("version", false, "")
	flag.BoolVar(showVersion, "v", false, "")

	flag.StringVar(
		&configs.AssetsDirAsStr, "assets-dir", "./assets/",
		configs.GetLocalizedMsg("meta_conf.AssetsDir", nil),
	)
	flag.BoolVar(
		&configs.PermitConfAlterInRuntime, "permit-conf-alter", true,
		configs.GetLocalizedMsg("meta_conf.PermitConfAlterInRuntime", nil),
	)
	flag.BoolVar(
		&configs.PermitSelfAlterInRuntime, "permit-self-alter", true,
		configs.GetLocalizedMsg("meta_conf.PermitSelfAlterInRuntime", nil),
	)
	flag.BoolVar(
		&configs.PermitShellAlterInRuntime, "permit-shell-alter", true,
		configs.GetLocalizedMsg("meta_conf.PermitShellAlterInRuntime", nil),
	)
	flag.BoolVar(
		&configs.PermitRemotePullInRuntime, "permit-remote-push", true,
		configs.GetLocalizedMsg("meta_conf.PermitRemotePullInRuntime", nil),
	)
	// TODO: What can be put as its IP Address?
	flag.StringVar(&configs.RemotePullSource, "remote-pull-source", ":65431", "")
	// TODO: require compatibility on different operating systems
	flag.StringVar(&configs.LocalRootCaCertAbsPath, "local-root-ca-cert-abs-path", "", "")
	flag.StringVar(&configs.LocalRootCaKeyAbsPath, "local-root-ca-key-abs-path", "", "")
	flag.StringVar(&configs.TrustedCertAbsPath, "trusted-ca-cert-abs-path", "", "")

	flag.Parse()
	if *showVersion {
		fmt.Printf("B0gus Ver: %s\n", versionStr)
		fmt.Printf("Build time: %s\n", buildTimeStr)
		fmt.Printf("Curr ISA: %s\n", runtime.GOARCH)
		fmt.Printf("Curr OS: %s\n", runtime.GOOS)
		os.Exit(0)
	} else if bogusConf == nil {
		configs.Logger.Fatal("unable to find setting up configuration!")
	}
}

/*-----------------------------------------------------------------------------------------------------
   Overview:
  [shell cmd] + +-> record_database
  	          | |
  	local conf+-+                     +--> SSH                   (almost there)
              | |       b0gus         |--> SMTP                  (basic shape)
   remote push+ +-> services manager -+--> FTP                   (draft)
  	                                  |--> NTP                   (almost there)
  	                                  |--> DNS                   (draft)
  	                                  |--> fake database         (not even a draft)
  	                                  +--> HTTP(s)               (not even a draft)
  	      							   ...
 *-----------------------------------------------------------------------------------------------------
 TO-Evaluate: configuration should not be located inside the environment where the program stays.
	configuration can fetch from network or filled by an interactive shell.
	but if so, there will be so many ways to set configuration that it becomes vagues
	to stop at one persistent state.
	Conceive that a shell will have to confirm if a concurrent and conflict push update can be applied.
 *-----------------------------------------------------------------------------------------------------
*/

// The entry of b0gus.
// configuration in `./configs/` should be properly set up before executing
func main() {
	defer func() { _ = configs.Logger.Sync() }()
	initConfFlags()
	db, dbStr, err := configs.SelectDatabaseBackend(
		&bogusConf.ServerConfig.RecDBConfig,
	)
	// Connect to Database. Create table when being ok to run the server
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"main.DatabaseConnectionError",
			map[string]any{
				"DatabaseStr": dbStr,
				"ErrInfo":     err.Error(),
			},
		)
		configs.Logger.Error(payload)
		return
	} else if db == nil {
		payload := configs.GetLocalizedMsg(
			"main.DatabaseEmptyError", nil,
		)
		configs.Logger.Error(payload)
		return
	}

	var (
		globRecordDb = configs.RuntimeDB{}
		terminator   = make(chan struct{}, 1)
		signalChan   = make(chan os.Signal, 1)
	)
	globRecordDb.AlterDatabaseHandler(db)

	// signal notification to terminate the whole server
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// configs.GlobConfigMan.Init()
	// configs.GlobConfigMan.Register(
	//	func(updConfig *configs.LocalConfig) {
	//		tmpDB, tmpDBStr, _err := configs.SelectDatabaseBackend(
	//			&updConfig.ServerConfig.RecDBConfig,
	//		)
	//		if tmpDB == nil || _err != nil || tmpDBStr == "" {
	//			payload := configs.GetLocalizedMsg(
	//				"main.DatabaseChangingWarn", nil,
	//			)
	//			configs.Logger.Warn(payload)
	//			return
	//		}
	//		globRecordDb.AlterDatabaseHandler(tmpDB)
	//	},
	// )
	go func() {
		sig := <-signalChan
		payload := configs.GetLocalizedMsg(
			"main.TerminationSignalWarn",
			map[string]any{"SignalStr": sig.String()},
		)
		configs.Logger.Warn(payload)
		terminator <- struct{}{}
		close(terminator)
		signal.Stop(signalChan)
		close(signalChan)
	}()
	services.Brancher(terminator, bogusConf, &globRecordDb)
	// configs.GlobConfigMan.SelfDestroy()
	close(configs.UpdateFlag)
}
