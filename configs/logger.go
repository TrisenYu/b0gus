package configs

import (
	"time"

	logrus "github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func init() {
	logrus.SetLevel(logrus.TraceLevel)
	// TODO: gain from global configuration and decide what log file to also
	// utilize as the output destination when necessary
	// logrus.SetOutput(io.MultiWriter(writer1, writer2))
	logrus.SetFormatter(
		&logrus.TextFormatter{
			TimestampFormat: time.StampMilli,
		},
	)
}
