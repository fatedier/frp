package log

import (
	"github.com/astaxie/beego/logs"
)

var Log *logs.BeeLogger

func init() {
	Log = logs.NewLogger(1000)
	Log.EnableFuncCallDepth(true)
	Log.SetLogFuncCallDepth(Log.GetLogFuncCallDepth() + 1)
}

func InitLog(logWay string, logFile string, logLevel string) {
	SetLogFile(logWay, logFile)
	SetLogLevel(logLevel)
}

// logWay: such as file or console
func SetLogFile(logWay string, logFile string) {
	if logWay == "console" {
		Log.SetLogger("console", "")
	} else {
		Log.SetLogger("file", `{"filename": "`+logFile+`"}`)
	}
}

// value: error, warning, info, debug
func SetLogLevel(logLevel string) {
	level := 4 // warning

	switch logLevel {
	case "error":
		level = 3
	case "warn":
		level = 4
	case "info":
		level = 6
	case "debug":
		level = 7
	default:
		level = 4
	}

	Log.SetLevel(level)
}

// wrap log
func Error(format string, v ...interface{}) {
	Log.Error(format, v...)
}

func Warn(format string, v ...interface{}) {
	Log.Warn(format, v...)
}

func Info(format string, v ...interface{}) {
	Log.Info(format, v...)
}

func Debug(format string, v ...interface{}) {
	Log.Debug(format, v...)
}
