// Package configs
package configs

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	loggerEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	writeSyncer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
	)
	LoggerCore := zapcore.NewCore(loggerEncoder, writeSyncer, zapcore.InfoLevel)
	Logger = zap.New(LoggerCore, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(Logger)
	// TODO: gain from global configuration and decide writing to which log file.
}
