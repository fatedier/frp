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
	"fmt"
	"io"
	"os"
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

type outputMode int

// DiscardNonColorEscSeq supports the divided color escape sequence.
// But non-color escape sequence is not output.
// Please use the OutputNonColorEscSeq If you want to output a non-color
// escape sequences such as ncurses. However, it does not support the divided
// color escape sequence.
const (
	_ outputMode = iota
	DiscardNonColorEscSeq
	OutputNonColorEscSeq
)

// NewAnsiColorWriter creates and initializes a new ansiColorWriter
// using io.Writer w as its initial contents.
// In the console of Windows, which change the foreground and background
// colors of the text by the escape sequence.
// In the console of other systems, which writes to w all text.
func NewAnsiColorWriter(w io.Writer) io.Writer {
	return NewModeAnsiColorWriter(w, DiscardNonColorEscSeq)
}

// NewModeAnsiColorWriter create and initializes a new ansiColorWriter
// by specifying the outputMode.
func NewModeAnsiColorWriter(w io.Writer, mode outputMode) io.Writer {
	if _, ok := w.(*ansiColorWriter); !ok {
		return &ansiColorWriter{
			w:    w,
			mode: mode,
		}
	}
	return w
}

const (
	y1  = `0123456789`
	y2  = `0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789`
	y3  = `0000000000111111111122222222223333333333444444444455555555556666666666777777777788888888889999999999`
	y4  = `0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789`
	mo1 = `000000000111`
	mo2 = `123456789012`
	d1  = `0000000001111111111222222222233`
	d2  = `1234567890123456789012345678901`
	h1  = `000000000011111111112222`
	h2  = `012345678901234567890123`
	mi1 = `000000000011111111112222222222333333333344444444445555555555`
	mi2 = `012345678901234567890123456789012345678901234567890123456789`
	s1  = `000000000011111111112222222222333333333344444444445555555555`
	s2  = `012345678901234567890123456789012345678901234567890123456789`
)

func formatTimeHeader(when time.Time) ([]byte, int) {
	y, mo, d := when.Date()
	h, mi, s := when.Clock()
	//len("2006/01/02 15:04:05 ")==20
	var buf [20]byte

	buf[0] = y1[y/1000%10]
	buf[1] = y2[y/100]
	buf[2] = y3[y-y/100*100]
	buf[3] = y4[y-y/100*100]
	buf[4] = '/'
	buf[5] = mo1[mo-1]
	buf[6] = mo2[mo-1]
	buf[7] = '/'
	buf[8] = d1[d-1]
	buf[9] = d2[d-1]
	buf[10] = ' '
	buf[11] = h1[h]
	buf[12] = h2[h]
	buf[13] = ':'
	buf[14] = mi1[mi]
	buf[15] = mi2[mi]
	buf[16] = ':'
	buf[17] = s1[s]
	buf[18] = s2[s]
	buf[19] = ' '

	return buf[0:], d
}

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})

	w32Green   = string([]byte{27, 91, 52, 50, 109})
	w32White   = string([]byte{27, 91, 52, 55, 109})
	w32Yellow  = string([]byte{27, 91, 52, 51, 109})
	w32Red     = string([]byte{27, 91, 52, 49, 109})
	w32Blue    = string([]byte{27, 91, 52, 52, 109})
	w32Magenta = string([]byte{27, 91, 52, 53, 109})
	w32Cyan    = string([]byte{27, 91, 52, 54, 109})

	reset = string([]byte{27, 91, 48, 109})
)

func ColorByStatus(cond bool, code int) string {
	switch {
	case code >= 200 && code < 300:
		return map[bool]string{true: green, false: w32Green}[cond]
	case code >= 300 && code < 400:
		return map[bool]string{true: white, false: w32White}[cond]
	case code >= 400 && code < 500:
		return map[bool]string{true: yellow, false: w32Yellow}[cond]
	default:
		return map[bool]string{true: red, false: w32Red}[cond]
	}
}

func ColorByMethod(cond bool, method string) string {
	switch method {
	case "GET":
		return map[bool]string{true: blue, false: w32Blue}[cond]
	case "POST":
		return map[bool]string{true: cyan, false: w32Cyan}[cond]
	case "PUT":
		return map[bool]string{true: yellow, false: w32Yellow}[cond]
	case "DELETE":
		return map[bool]string{true: red, false: w32Red}[cond]
	case "PATCH":
		return map[bool]string{true: green, false: w32Green}[cond]
	case "HEAD":
		return map[bool]string{true: magenta, false: w32Magenta}[cond]
	case "OPTIONS":
		return map[bool]string{true: white, false: w32White}[cond]
	default:
		return reset
	}
}

// Guard Mutex to guarantee atomicity of W32Debug(string) function
var mu sync.Mutex

// Helper method to output colored logs in Windows terminals
func W32Debug(msg string) {
	mu.Lock()
	defer mu.Unlock()

	current := time.Now()
	w := NewAnsiColorWriter(os.Stdout)

	fmt.Fprintf(w, "[beego] %v %s\n", current.Format("2006/01/02 - 15:04:05"), msg)
}
