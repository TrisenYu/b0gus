package configs

import (
	"time"

	logrus "github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func init() {
	logrus.SetFormatter(
		&logrus.TextFormatter{
			TimestampFormat: time.StampMilli,
		},
	)
}
