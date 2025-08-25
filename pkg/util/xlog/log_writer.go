// Copyright 2025 The frp Authors
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

package xlog

import "strings"

// LogWriter forwards writes to frp's logger at configurable level.
// It is safe for concurrent use as long as the underlying Logger is thread-safe.
type LogWriter struct {
	xl      *Logger
	logFunc func(string)
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	w.logFunc(msg)
	return len(p), nil
}

func NewTraceWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Tracef("%s", msg) },
	}
}

func NewDebugWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Debugf("%s", msg) },
	}
}

func NewInfoWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Infof("%s", msg) },
	}
}

func NewWarnWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Warnf("%s", msg) },
	}
}

func NewErrorWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Errorf("%s", msg) },
	}
}
