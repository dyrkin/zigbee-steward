package logger

import (
	"github.com/op/go-logging"
	"os"
)

type Logger struct {
	*logging.Logger
}

func MustGetLogger(module string) *Logger {
	log := logging.MustGetLogger(module)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{level:.5s} %{module} %{color:reset} %{message}`,
	)

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLeveled := logging.AddModuleLevel(backendFormatter)
	backendLeveled.SetLevel(logging.DEBUG, "")

	logging.SetBackend(backendLeveled)
	return &Logger{log}
}

func (log *Logger) Debugf(format string, args ...interface{}) {
	if log.IsEnabledFor(logging.DEBUG) {
		renderLazyArgs(args...)
		log.Logger.Debugf(format, args...)
	}
}

func (log *Logger) Infof(format string, args ...interface{}) {
	if log.IsEnabledFor(logging.INFO) {
		renderLazyArgs(args...)
		log.Logger.Infof(format, args...)
	}
}

func (log *Logger) Info(args ...interface{}) {
	if log.IsEnabledFor(logging.INFO) {
		renderLazyArgs(args...)
		log.Logger.Info(args...)
	}
}

func (log *Logger) Errorf(format string, args ...interface{}) {
	if log.IsEnabledFor(logging.ERROR) {
		renderLazyArgs(args...)
		log.Logger.Errorf(format, args...)
	}
}

func (log *Logger) Error(args ...interface{}) {
	if log.IsEnabledFor(logging.ERROR) {
		renderLazyArgs(args...)
		log.Logger.Error(args...)
	}
}

func renderLazyArgs(args ...interface{}) {
	for i := range args {
		if fn, ok := args[i].(func() string); ok {
			args[i] = fn()
		}
	}
}
