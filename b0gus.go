package main

/// Last modified at 2026/04/17 星期五 21:26:16
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
	versionStr   string
	buildTimeStr string
	hashValStr   string
	builtByStr   string
)

func initConfFlags() {
	flag.StringVar(
		&configs.LocalConfigPathAsStr, "conf-path", "./configs/config.toml",
		configs.GetLocalizedMsg("meta_conf.ConfPath", nil),
	)
	configs.GlobConf.Store(configs.LoadDefaultConfig(""))
	f := flag.Lookup("conf-path")
	if f != nil { // reset the configuration path description if feasible
		f.Usage = configs.GetLocalizedMsg("meta_conf.ConfPath", nil)
	}
	showVersion := flag.Bool("version", false, "")
	flag.BoolVar(showVersion, "v", false, "")

	flag.StringVar(
		&configs.AssetsDirAsStr, "assets-dir", "./assets/",
		configs.GetLocalizedMsg("meta_conf.AssetsDir", nil),
	)
	flag.StringVar(&configs.RemotePullSource, "remote-pull-source", ":65431", "")
	flag.StringVar(
		&configs.LocalRootCaCertAbsPath, "local-root-ca-cert-abs-path",
		"./configs/root-ca.cert", "",
	)
	flag.StringVar(
		&configs.LocalRootCaKeyAbsPath, "local-root-ca-key-abs-path",
		"./configs/root-ca.key", "",
	)

	flag.Parse()
	if *showVersion {
		payload := `b0gus Version: %s-%s
Build Time:    %s
Build Hash:    %s
Builder Name:  %s
Current ISA:   %s
Current OS:    %s
`
		fmt.Printf(
			payload, versionStr, configs.BuildTypeStr,
			buildTimeStr, hashValStr, builtByStr,
			runtime.GOARCH, runtime.GOOS,
		)
		os.Exit(0)
	} else if configs.GlobConf.Load() == nil {
		payload := configs.GetLocalizedMsg("main.FailToApplyConfiguration", nil)
		configs.Logger.Fatal(payload)
	}
}

/*-------------------------------------------------------------------------------------------------+
   Overview:
                +-> recording database
                |
  	local conf+-+                     +--> SSH                   (almost there)
              | |       b0gus         |--> SMTP                  (basic shape)
   remote push+ +-> services manager -+--> FTP                   (not even a draft, require sandbox)
  	          |                       |--> NTP                   (almost there)
  [shell cmd] +                       |--> DNS                   (basic shape)
  	                                  |--> fake database         (not even a draft)
  	                                  +--> HTTP(s)               (basic shape)
  	      							   ...
                                       ^
                                   TODO: decouple this layer from current computer by Secure RPC?
 *-------------------------------------------------------------------------------------------------+
 TO-Evaluate: configuration should not be located inside the environment where the program stays.
	configuration can fetch from network or filled by an interactive shell.
	but if so, there will be so many ways to set configuration that it becomes vagues
	to stop at one persistent state. Conceive that a shell will have to confirm
	if a concurrent and conflict push update can be applied.
 *-------------------------------------------------------------------------------------------------+
*/

// The entry of b0gus.
// configuration in `./configs/` should be properly set up before executing
func main() {
	initConfFlags()
	defer func() { _ = configs.Logger.Sync() }()
	db, dbStr, err := configs.SelectDatabaseBackend(
		&configs.GlobConf.Load().ServerConfig.RecDBConfig,
	)
	if err != nil {
		configs.Logger.Error(configs.GetLocalizedMsg(
			"main.DatabaseConnectionError",
			map[string]any{
				"DatabaseStr": dbStr,
				"ErrInfo":     err.Error(),
			},
		))
		return
	} else if db == nil {
		configs.Logger.Error(configs.GetLocalizedMsg(
			"main.DatabaseEmptyError", nil,
		))
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
	go func() {
		sig := <-signalChan
		configs.Logger.Warn(configs.GetLocalizedMsg(
			"main.TerminationSignalWarn",
			map[string]any{"SignalStr": sig.String()},
		))
		terminator <- struct{}{}
		close(terminator)
		signal.Stop(signalChan)
		close(signalChan)
	}()
	services.Brancher(terminator, &configs.GlobConf, &globRecordDb)
}
