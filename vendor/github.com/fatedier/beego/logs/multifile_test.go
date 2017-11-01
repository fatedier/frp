// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestFiles_1(t *testing.T) {
	log := NewLogger(10000)
	log.SetLogger("multifile", `{"filename":"test.log","separate":["emergency", "alert", "critical", "error", "warning", "notice", "info", "debug"]}`)
	log.Debug("debug")
	log.Informational("info")
	log.Notice("notice")
	log.Warning("warning")
	log.Error("error")
	log.Alert("alert")
	log.Critical("critical")
	log.Emergency("emergency")
	fns := []string{""}
	fns = append(fns, levelNames[0:]...)
	name := "test"
	suffix := ".log"
	for _, fn := range fns {

		file := name + suffix
		if fn != "" {
			file = name + "." + fn + suffix
		}
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		b := bufio.NewReader(f)
		lineNum := 0
		lastLine := ""
		for {
			line, _, err := b.ReadLine()
			if err != nil {
				break
			}
			if len(line) > 0 {
				lastLine = string(line)
				lineNum++
			}
		}
		var expected = 1
		if fn == "" {
			expected = LevelDebug + 1
		}
		if lineNum != expected {
			t.Fatal(file, "has", lineNum, "lines not "+strconv.Itoa(expected)+" lines")
		}
		if lineNum == 1 {
			if !strings.Contains(lastLine, fn) {
				t.Fatal(file + " " + lastLine + " not contains the log msg " + fn)
			}
		}
		os.Remove(file)
	}

}
