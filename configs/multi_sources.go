// Package configs
// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package configs

import (
	"context"
	"crypto/tls"
	"fmt"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// LoadDefaultConfig will load configuration from LocalConfigPathAsStr,
// which is set as `configs/config.toml` by default.
func LoadDefaultConfig(confPath string) *LocalConfig {
	if confPath == "" {
		confPath = LocalConfigPathAsStr
	}
	viper.SetConfigFile(confPath)
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		payload := GetLocalizedMsg(
			"configs.ReadingConfigurationFailure",
			map[string]any{"ErrInfo": err},
		)
		Logger.Fatal(payload)
		return nil
	}

	// TODO: abstract a configuration layer for pushing updates by different sources.
	// 	don't really have to stick with one default changeable toml file.
	// 	Shell command can choose to disable the synchronous pulling
	//  otherwise implementation for a confirm mechanism is necessary

	err = viper.Unmarshal(&currConfig)
	if err != nil {
		payload := GetLocalizedMsg(
			"configs.UnmarshallingConfigurationFailure",
			map[string]any{"ErrInfo": err},
		)
		Logger.Fatal(payload)
		return nil
	}
	return &currConfig
}

// AlterLocalConf will be invoked by monitorGivenLocalConf if the global configuration is enabled
func AlterLocalConf() {
	confLock.Lock()
	defer confLock.Unlock()
	err := viper.Unmarshal(&tmpHotConf)
	if err != nil {
		payload := GetLocalizedMsg(
			"configs.ReloadingConfigurationFailure",
			map[string]any{"ErrInfo": err},
		)
		Logger.Error(payload)
		return
	}
	UpdateFlag <- tmpHotConf
	currConfig = *tmpHotConf
}

// monitorGivenLocalConf
func monitorGivenLocalConf(ctx context.Context, path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		Logger.Error("unable to create watcher for given file!")
		return
	}
	defer func() { _ = watcher.Close() }()
	fileName := filepath.Base(path)
	if err := watcher.Add(filepath.Dir(path)); err != nil {
		Logger.Error("unable to monitor the directory of given file!")
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok || filepath.Base(event.Name) != fileName {
				continue
			}
			switch event.Op {
			case fsnotify.Create:
				fallthrough
			case fsnotify.Write:
				// check and push current update
				AlterLocalConf()
			case fsnotify.Remove:
				Logger.Info("given file has been removed at time, won't reload!")
				fallthrough
			default:
				// do not concern rename event.
				continue
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				payload := fmt.Sprintf(
					"errors happened on file watcher! ErrInfo: %v", err,
				)
				Logger.Error(payload)
			}
			continue
		// TODO: updates from networking-end
		}
	}
}

// network updates should enable augmented authentication

func PullUpdatesFromRemote() {
	// listen at local port and obey some formal syntax/private protocol
	// during the runtime, the monitored port might be altered to another port
	// so the session

	// notice that this interface will provide distributed communication ability
	// so encryption is required

	// As an TLS client or server?
	tlsConn, err := tls.Dial("tcp", RemotePullSource, &tls.Config{})
	if err != nil || tlsConn == nil {
		// can not connect to remote pulling source.
		return
	}
	// TODO what if the connection sends a broken configuration?
	// tlsConn.Read()
	_ = tlsConn.Close()
}

func DetectConfigUpdates(ctx context.Context, filepath string) {
	// crying stack
	go monitorGivenLocalConf(ctx, filepath)

}
