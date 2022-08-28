// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"

	"github.com/fatedier/beego/logs"
)

// Log is the under log object
var Log *logs.BeeLogger

func init() {
	Log = logs.NewLogger(200)
	Log.EnableFuncCallDepth(true)
	Log.SetLogFuncCallDepth(Log.GetLogFuncCallDepth() + 1)
}

func InitLog(logWay string, logFile string, logLevel string, maxdays int64, disableLogColor bool) {
	SetLogFile(logWay, logFile, maxdays, disableLogColor)
	SetLogLevel(logLevel)
}

// SetLogFile to configure log params
// logWay: file or console
func SetLogFile(logWay string, logFile string, maxdays int64, disableLogColor bool) {
	if logWay == "console" {
		params := ""
		if disableLogColor {
			params = `{"color": false}`
		}
		_ = Log.SetLogger("console", params)
	} else {
		params := fmt.Sprintf(`{"filename": "%s", "maxdays": %d}`, logFile, maxdays)
		_ = Log.SetLogger("file", params)
	}
}

// SetLogLevel set log level, default is warning
// value: error, warning, info, debug, trace
func SetLogLevel(logLevel string) {
	var level int
	switch logLevel {
	case "error":
		level = 3
	case "warn":
		level = 4
	case "info":
		level = 6
	case "debug":
		level = 7
	case "trace":
		level = 8
	default:
		level = 4 // warning
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

func Trace(format string, v ...interface{}) {
	Log.Trace(format, v...)
}
