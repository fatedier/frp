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
	"cmp"
	"slices"

	"github.com/fatedier/frp/pkg/util/log"
)

type LogPrefix struct {
	// Name is the name of the prefix, it won't be displayed in log but used to identify the prefix.
	Name string
	// Value is the value of the prefix, it will be displayed in log.
	Value string
	// The prefix with higher priority will be displayed first, default is 10.
	Priority int
}

// Logger is not thread safety for operations on prefix
type Logger struct {
	prefixes []LogPrefix

	prefixString string
}

func New() *Logger {
	return &Logger{
		prefixes: make([]LogPrefix, 0),
	}
}

func (l *Logger) ResetPrefixes() (old []LogPrefix) {
	old = l.prefixes
	l.prefixes = make([]LogPrefix, 0)
	l.prefixString = ""
	return
}

func (l *Logger) AppendPrefix(prefix string) *Logger {
	return l.AddPrefix(LogPrefix{
		Name:     prefix,
		Value:    prefix,
		Priority: 10,
	})
}

func (l *Logger) AddPrefix(prefix LogPrefix) *Logger {
	found := false
	if prefix.Priority <= 0 {
		prefix.Priority = 10
	}
	for _, p := range l.prefixes {
		if p.Name == prefix.Name {
			found = true
			p.Value = prefix.Value
			p.Priority = prefix.Priority
		}
	}
	if !found {
		l.prefixes = append(l.prefixes, prefix)
	}
	l.renderPrefixString()
	return l
}

func (l *Logger) renderPrefixString() {
	slices.SortStableFunc(l.prefixes, func(a, b LogPrefix) int {
		return cmp.Compare(a.Priority, b.Priority)
	})
	l.prefixString = ""
	for _, v := range l.prefixes {
		l.prefixString += "[" + v.Value + "] "
	}
}

func (l *Logger) Spawn() *Logger {
	nl := New()
	nl.prefixes = append(nl.prefixes, l.prefixes...)
	nl.renderPrefixString()
	return nl
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	log.Logger.Errorf(l.prefixString+format, v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	log.Logger.Warnf(l.prefixString+format, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	log.Logger.Infof(l.prefixString+format, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	log.Logger.Debugf(l.prefixString+format, v...)
}

func (l *Logger) Tracef(format string, v ...interface{}) {
	log.Logger.Tracef(l.prefixString+format, v...)
}
