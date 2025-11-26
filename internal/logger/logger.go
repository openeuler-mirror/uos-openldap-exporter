package logger

import (
	"github.com/sirupsen/logrus"
)

// New creates a new logger instance with the specified log level
func New(levelStr string) *logrus.Logger {
	log := logrus.New()
	
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		level = logrus.InfoLevel
	}
	
	log.SetLevel(level)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	
	return log
}