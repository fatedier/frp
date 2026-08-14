// Copyright 2026 The frp Authors
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
	"bytes"
	"testing"

	goliblog "github.com/fatedier/golib/log"
	"github.com/stretchr/testify/require"

	frplog "github.com/fatedier/frp/pkg/util/log"
)

func TestPrefixIsNotPartOfFormatString(t *testing.T) {
	tests := []struct {
		name string
		log  func(*Logger)
	}{
		{name: "error", log: func(xl *Logger) { xl.Errorf("value [%s]", "ok") }},
		{name: "warn", log: func(xl *Logger) { xl.Warnf("value [%s]", "ok") }},
		{name: "info", log: func(xl *Logger) { xl.Infof("value [%s]", "ok") }},
		{name: "debug", log: func(xl *Logger) { xl.Debugf("value [%s]", "ok") }},
		{name: "trace", log: func(xl *Logger) { xl.Tracef("value [%s]", "ok") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogs(t)

			tt.log(New().AppendPrefix("%1000000s"))

			require.Contains(t, output.String(), "[%1000000s] value [ok]")
			require.Less(t, output.Len(), 1024)
		})
	}
}

func TestFormattingSemanticsArePreserved(t *testing.T) {
	output := captureLogs(t)
	xl := New().AppendPrefix("run")

	xl.Infof("%[2]s %[1]s", "first", "second")
	xl.Infof("100% complete")

	require.Contains(t, output.String(), "[run] second first")
	require.Contains(t, output.String(), "[run] 100% complete")
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	output := bytes.NewBuffer(nil)
	oldLogger := frplog.Logger
	frplog.Logger = goliblog.New(
		goliblog.WithOutput(output),
		goliblog.WithLevel(goliblog.TraceLevel),
		goliblog.WithCaller(false),
	)
	t.Cleanup(func() {
		frplog.Logger = oldLogger
	})
	return output
}
