// Package configs
// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package configs

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	currConfig = LocalConfig{}
	tmpHotConf = &LocalConfig{}
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
	err := viper.Unmarshal(&tmpHotConf)
	if err != nil {
		payload := GetLocalizedMsg(
			"configs.ReloadingConfigurationFailure",
			map[string]any{"ErrInfo": err},
		)
		Logger.Error(payload)
		return
	}
	// UpdateFlag <- tmpHotConf
	// currConfig = *tmpHotConf
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
			// we don't have to listen at current time due to cancel function is called.
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
				// AlterLocalConf()
			case fsnotify.Remove:
				Logger.Info("given file has been removed at time, won't reload!")
				fallthrough
			default:
				// do not concern rename event.
				continue
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				var sb strings.Builder
				sb.WriteString("errors happened on file watcher! ErrInfo: ")
				sb.WriteString(err.Error())
				Logger.Error(sb.String())
			}
			continue
			// TODO: updates from networking-end and cli-shell end.
		}
	}
}

// network updates should enable augmented authentication

func DetectConfigUpdates(ctx context.Context, filepath string) {
	// crying stack
	go monitorGivenLocalConf(ctx, filepath)

}
