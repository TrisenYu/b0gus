// Package configs
package configs

/// Last modified at 2026/05/09 星期六 23:39:42
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger       *zap.Logger
	BuildTypeStr string
)

func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	loggerEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	writeSyncer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
	)
	var choice = zap.InfoLevel
	if strings.Contains(BuildTypeStr, "debug") {
		choice = zap.DebugLevel
	}
	LoggerCore := zapcore.NewCore(loggerEncoder, writeSyncer, choice)
	Logger = zap.New(
		LoggerCore,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	zap.ReplaceGlobals(Logger)
	// TODO: gain from global configuration and decide writing to which log file.
}
