package main

/// Last modified at 2026/05/16 星期六 12:48:34
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"os"
	"os/signal"
	"syscall"

	"b0gus/configs"
	"b0gus/services"
)

// [TODO]: rpc registering

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
                               [TODO]: decouple this layer from current host by using Encrypted RPC?
 *-------------------------------------------------------------------------------------------------+
 TO-Evaluate: configuration should not be located inside the environment where the program stays.
	configuration can fetch from network or filled by an interactive shell.
	but if so, there will be so many ways to set configuration that it becomes vagues
	to stop at one persistent state. Conceive that a shell will have to confirm
	if a concurrent and conflict push update can be applied.
 *-------------------------------------------------------------------------------------------------+
*/

// main function is the entry of b0gus.
// configuration in `./configs/` should be properly set up before executing.
// At present, b0gus run locally
func main() {
	configs.InitConfFlags()
	defer configs.SetLogger(nil)
	terminator := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)

	// signal notification to terminate the whole server
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalChan
		configs.Logger().Warn(configs.GetLocalizedMsg(
			"main.TerminationSignalWarn",
			map[string]any{"SignalStr": sig.String()},
		))
		terminator <- struct{}{}
		signal.Stop(signalChan)
		close(signalChan)
		close(terminator)
	}()
	services.Brancher(terminator, &configs.GlobConf)
}
