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
	"errors"
	"github.com/fatedier/frp/pkg/util/log/events"
	"github.com/fatedier/frp/pkg/util/system"
	"github.com/fatedier/golib/log"
	"golang.org/x/sys/windows/svc/eventlog"
	"strings"
	"time"
)

var eventWriter *EventWriter = nil

type EventWriter struct {
	logInstance *eventlog.Log
}

func (e EventWriter) Write(p []byte) (n int, err error) {
	// Write at default log level and current time (fallback)
	return e.WriteLog(p, log.InfoLevel, time.Now())
}

func (e EventWriter) WriteLog(p []byte, level log.Level, _ time.Time) (n int, err error) {
	var eid uint32 = events.Undefined
	s := strings.TrimSpace(string(p))
	switch level {
	case log.TraceLevel:
		fallthrough
	case log.DebugLevel:
		eid = events.Undefined
		fallthrough
	case log.InfoLevel:
		eid = events.InfoUndefined
		err := e.logInstance.Info(eid, s)
		if err != nil {
			return 0, err
		}
		return len(s), nil
	case log.WarnLevel:
		eid = events.WarnUndefined
		err := e.logInstance.Warning(eid, s)
		if err != nil {
			return 0, err
		}
		return len(s), nil
	case log.ErrorLevel:
		eid = events.ErrUndefined
		err := e.logInstance.Error(eid, s)
		if err != nil {
			return 0, err
		}
		return len(s), nil
	default:
		// This should not happen
		return 0, errors.ErrUnsupported
	}
}

// GetEventWriter returns current EventWriter instance.
// Returns nil if not successfully initialized.
func GetEventWriter() *EventWriter {
	return eventWriter
}

// InitEventWriter tries initializing an EventWriter instance.
func InitEventWriter() error {
	if eventWriter != nil {
		return nil
	}
	logInstance, err := eventlog.Open(system.ServiceName)
	if err != nil {
		return err
	}
	eventWriter = &EventWriter{
		logInstance: logInstance,
	}
	return nil
}

// DestroyEventWriter closes the EventWriter instance.
// This should be called in defer.
func DestroyEventWriter() error {
	if eventWriter == nil {
		return nil
	}
	defer func() {
		eventWriter = nil
	}()
	return eventWriter.logInstance.Close()
}