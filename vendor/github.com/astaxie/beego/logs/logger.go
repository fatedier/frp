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
	"io"
	"sync"
	"time"
)

type logWriter struct {
	sync.Mutex
	writer io.Writer
}

func newLogWriter(wr io.Writer) *logWriter {
	return &logWriter{writer: wr}
}

func (lg *logWriter) println(when time.Time, msg string) {
	lg.Lock()
	h, _ := formatTimeHeader(when)
	lg.writer.Write(append(append(h, msg...), '\n'))
	lg.Unlock()
}

func formatTimeHeader(when time.Time) ([]byte, int) {
	y, mo, d := when.Date()
	h, mi, s := when.Clock()
	//len(2006/01/02 15:03:04)==19
	var buf [20]byte
	t := 3
	for y >= 10 {
		p := y / 10
		buf[t] = byte('0' + y - p*10)
		y = p
		t--
	}
	buf[0] = byte('0' + y)
	buf[4] = '/'
	if mo > 9 {
		buf[5] = '1'
		buf[6] = byte('0' + mo - 9)
	} else {
		buf[5] = '0'
		buf[6] = byte('0' + mo)
	}
	buf[7] = '/'
	t = d / 10
	buf[8] = byte('0' + t)
	buf[9] = byte('0' + d - t*10)
	buf[10] = ' '
	t = h / 10
	buf[11] = byte('0' + t)
	buf[12] = byte('0' + h - t*10)
	buf[13] = ':'
	t = mi / 10
	buf[14] = byte('0' + t)
	buf[15] = byte('0' + mi - t*10)
	buf[16] = ':'
	t = s / 10
	buf[17] = byte('0' + t)
	buf[18] = byte('0' + s - t*10)
	buf[19] = ' '

	return buf[0:], d
}
