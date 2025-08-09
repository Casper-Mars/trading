package logger

import (
	"io"
	"os"

	"data-collection-system/pkg/config"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init 初始化日志组件
func Init(cfg config.LogConfig) {
	log = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// 设置日志格式
	switch cfg.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// 设置输出目标
	switch cfg.Output {
	case "file":
		file, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.SetOutput(os.Stdout)
		} else {
			log.SetOutput(io.MultiWriter(os.Stdout, file))
		}
	default:
		log.SetOutput(os.Stdout)
	}
}

// Debug 输出调试级别日志
func Debug(args ...interface{}) {
	if log != nil {
		log.Debug(args...)
	}
}

// Debugf 输出格式化调试级别日志
func Debugf(format string, args ...interface{}) {
	if log != nil {
		log.Debugf(format, args...)
	}
}

// Info 输出信息级别日志
func Info(args ...interface{}) {
	if log != nil {
		log.Info(args...)
	}
}

// Infof 输出格式化信息级别日志
func Infof(format string, args ...interface{}) {
	if log != nil {
		log.Infof(format, args...)
	}
}

// Warn 输出警告级别日志
func Warn(args ...interface{}) {
	if log != nil {
		log.Warn(args...)
	}
}

// Warnf 输出格式化警告级别日志
func Warnf(format string, args ...interface{}) {
	if log != nil {
		log.Warnf(format, args...)
	}
}

// Error 输出错误级别日志
func Error(args ...interface{}) {
	if log != nil {
		log.Error(args...)
	}
}

// Errorf 输出格式化错误级别日志
func Errorf(format string, args ...interface{}) {
	if log != nil {
		log.Errorf(format, args...)
	}
}

// Fatal 输出致命错误级别日志并退出程序
func Fatal(args ...interface{}) {
	if log != nil {
		log.Fatal(args...)
	}
}

// Fatalf 输出格式化致命错误级别日志并退出程序
func Fatalf(format string, args ...interface{}) {
	if log != nil {
		log.Fatalf(format, args...)
	}
}

// WithField 添加字段
func WithField(key string, value interface{}) *logrus.Entry {
	if log != nil {
		return log.WithField(key, value)
	}
	return nil
}

// WithFields 添加多个字段
func WithFields(fields logrus.Fields) *logrus.Entry {
	if log != nil {
		return log.WithFields(fields)
	}
	return nil
}