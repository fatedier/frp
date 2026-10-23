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
	"sync"
	"sync/atomic"

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

// Logger supports concurrent logging and prefix updates.
// A Logger must not be copied after first use.
type Logger struct {
	mu       sync.Mutex
	prefixes []LogPrefix

	prefixString atomic.Pointer[string]
}

func New() *Logger {
	return &Logger{
		prefixes: make([]LogPrefix, 0),
	}
}

func (l *Logger) ResetPrefixes() (old []LogPrefix) {
	l.mu.Lock()
	defer l.mu.Unlock()
	old = l.prefixes
	l.prefixes = make([]LogPrefix, 0)
	l.prefixString.Store(nil)
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
	l.mu.Lock()
	defer l.mu.Unlock()
	found := false
	if prefix.Priority <= 0 {
		prefix.Priority = 10
	}
	for i, p := range l.prefixes {
		if p.Name == prefix.Name {
			found = true
			l.prefixes[i].Value = prefix.Value
			l.prefixes[i].Priority = prefix.Priority
			break
		}
	}
	if !found {
		l.prefixes = append(l.prefixes, prefix)
	}
	l.renderPrefixString()
	return l
}

// The caller must hold l.mu or exclusively own an unpublished logger.
func (l *Logger) renderPrefixString() {
	slices.SortStableFunc(l.prefixes, func(a, b LogPrefix) int {
		return cmp.Compare(a.Priority, b.Priority)
	})
	var prefixString string
	for _, v := range l.prefixes {
		prefixString += "[" + v.Value + "] "
	}
	l.prefixString.Store(&prefixString)
}

func (l *Logger) Spawn() *Logger {
	nl := New()
	l.mu.Lock()
	nl.prefixes = append(nl.prefixes, l.prefixes...)
	l.mu.Unlock()
	nl.renderPrefixString()
	return nl
}

func (l *Logger) getPrefixString() string {
	if prefix := l.prefixString.Load(); prefix != nil {
		return *prefix
	}
	return ""
}

func (l *Logger) Errorf(format string, v ...any) {
	log.Logger.WithPrefix(l.getPrefixString()).Errorf(format, v...)
}

func (l *Logger) Warnf(format string, v ...any) {
	log.Logger.WithPrefix(l.getPrefixString()).Warnf(format, v...)
}

func (l *Logger) Infof(format string, v ...any) {
	log.Logger.WithPrefix(l.getPrefixString()).Infof(format, v...)
}

func (l *Logger) Debugf(format string, v ...any) {
	log.Logger.WithPrefix(l.getPrefixString()).Debugf(format, v...)
}

func (l *Logger) Tracef(format string, v ...any) {
	log.Logger.WithPrefix(l.getPrefixString()).Tracef(format, v...)
}
