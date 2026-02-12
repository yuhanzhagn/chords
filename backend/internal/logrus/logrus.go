package logrus

import (
	"os"

	"github.com/sirupsen/logrus"
)

func InitLogrus() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	return log
}
