package logger

import (
	"github.com/sirupsen/logrus"
)

// New creates a new logger instance with the specified log level and format
func New(levelStr string, format string) *logrus.Logger {
	log := logrus.New()

	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		level = logrus.InfoLevel
	}

	log.SetLevel(level)

	// 根据配置设置日志格式
	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return log
}
