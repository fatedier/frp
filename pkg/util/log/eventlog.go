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

//go:build !windows

package log

import (
	"github.com/fatedier/golib/log"
	"time"
)

type EventWriter struct {
}

func (e EventWriter) Write(p []byte) (n int, err error) {
	return log.DefaultWriter.Write(b)
}

func (e EventWriter) WriteLog(p []byte, _ log.Level, _ time.Time) (n int, err error) {
	return e.Write(p)
}

// GetEventWriter returns current EventWriter instance.
// Returns nil if not initialized.
func GetEventWriter() *EventWriter {
	return nil
}

// InitEventWriter tries initializing an EventWriter instance.
func InitEventWriter() {
}

// DestroyEventWriter closes the EventWriter instance.
// This should be called in defer.
func DestroyEventWriter() error {
	return nil
}