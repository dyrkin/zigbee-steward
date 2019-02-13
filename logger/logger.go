package logger

import (
	"github.com/op/go-logging"
	"os"
)

var log = logging.MustGetLogger("main")

func init() {
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{level:.5s} %{color:reset} %{message}`,
	)

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLeveled := logging.AddModuleLevel(backendFormatter)
	backendLeveled.SetLevel(logging.DEBUG, "")

	logging.SetBackend(backendLeveled)
}

func Debugf(format string, args ...interface{}) {
	if log.IsEnabledFor(logging.DEBUG) {
		renderLazyArgs(args...)
		log.Debugf(format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	if log.IsEnabledFor(logging.INFO) {
		renderLazyArgs(args...)
		log.Infof(format, args...)
	}
}

func Info(args ...interface{}) {
	if log.IsEnabledFor(logging.INFO) {
		renderLazyArgs(args...)
		log.Info(args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if log.IsEnabledFor(logging.ERROR) {
		renderLazyArgs(args...)
		log.Errorf(format, args...)
	}
}

func Error(args ...interface{}) {
	if log.IsEnabledFor(logging.ERROR) {
		renderLazyArgs(args...)
		log.Error(args...)
	}
}

func renderLazyArgs(args ...interface{}) {
	for i := range args {
		if fn, ok := args[i].(func() string); ok {
			args[i] = fn()
		}
	}
}
