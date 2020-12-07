// Copyright 2019 fatedier, fatedier@gmail.com
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

import (
	"github.com/fatedier/frp/pkg/util/log"
)

// Logger is not thread safety for operations on prefix
type Logger struct {
	prefixes []string

	prefixString string
}

func New() *Logger {
	return &Logger{
		prefixes: make([]string, 0),
	}
}

func (l *Logger) ResetPrefixes() (old []string) {
	old = l.prefixes
	l.prefixes = make([]string, 0)
	l.prefixString = ""
	return
}

func (l *Logger) AppendPrefix(prefix string) *Logger {
	l.prefixes = append(l.prefixes, prefix)
	l.prefixString += "[" + prefix + "] "
	return l
}

func (l *Logger) Spawn() *Logger {
	nl := New()
	for _, v := range l.prefixes {
		nl.AppendPrefix(v)
	}
	return nl
}

func (l *Logger) Error(format string, v ...interface{}) {
	log.Log.Error(l.prefixString+format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	log.Log.Warn(l.prefixString+format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	log.Log.Info(l.prefixString+format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	log.Log.Debug(l.prefixString+format, v...)
}

func (l *Logger) Trace(format string, v ...interface{}) {
	log.Log.Trace(l.prefixString+format, v...)
}
