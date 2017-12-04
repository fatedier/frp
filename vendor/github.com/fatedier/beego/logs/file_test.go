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
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestFilePerm(t *testing.T) {
	log := NewLogger(10000)
	// use 0666 as test perm cause the default umask is 022
	log.SetLogger("file", `{"filename":"test.log", "perm": "0666"}`)
	log.Debug("debug")
	log.Informational("info")
	log.Notice("notice")
	log.Warning("warning")
	log.Error("error")
	log.Alert("alert")
	log.Critical("critical")
	log.Emergency("emergency")
	file, err := os.Stat("test.log")
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode() != 0666 {
		t.Fatal("unexpected log file permission")
	}
	os.Remove("test.log")
}

func TestFile1(t *testing.T) {
	log := NewLogger(10000)
	log.SetLogger("file", `{"filename":"test.log"}`)
	log.Debug("debug")
	log.Informational("info")
	log.Notice("notice")
	log.Warning("warning")
	log.Error("error")
	log.Alert("alert")
	log.Critical("critical")
	log.Emergency("emergency")
	f, err := os.Open("test.log")
	if err != nil {
		t.Fatal(err)
	}
	b := bufio.NewReader(f)
	lineNum := 0
	for {
		line, _, err := b.ReadLine()
		if err != nil {
			break
		}
		if len(line) > 0 {
			lineNum++
		}
	}
	var expected = LevelDebug + 1
	if lineNum != expected {
		t.Fatal(lineNum, "not "+strconv.Itoa(expected)+" lines")
	}
	os.Remove("test.log")
}

func TestFile2(t *testing.T) {
	log := NewLogger(10000)
	log.SetLogger("file", fmt.Sprintf(`{"filename":"test2.log","level":%d}`, LevelError))
	log.Debug("debug")
	log.Info("info")
	log.Notice("notice")
	log.Warning("warning")
	log.Error("error")
	log.Alert("alert")
	log.Critical("critical")
	log.Emergency("emergency")
	f, err := os.Open("test2.log")
	if err != nil {
		t.Fatal(err)
	}
	b := bufio.NewReader(f)
	lineNum := 0
	for {
		line, _, err := b.ReadLine()
		if err != nil {
			break
		}
		if len(line) > 0 {
			lineNum++
		}
	}
	var expected = LevelError + 1
	if lineNum != expected {
		t.Fatal(lineNum, "not "+strconv.Itoa(expected)+" lines")
	}
	os.Remove("test2.log")
}

func TestFileRotate_01(t *testing.T) {
	log := NewLogger(10000)
	log.SetLogger("file", `{"filename":"test3.log","maxlines":4}`)
	log.Debug("debug")
	log.Info("info")
	log.Notice("notice")
	log.Warning("warning")
	log.Error("error")
	log.Alert("alert")
	log.Critical("critical")
	log.Emergency("emergency")
	rotateName := "test3" + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), 1) + ".log"
	b, err := exists(rotateName)
	if !b || err != nil {
		os.Remove("test3.log")
		t.Fatal("rotate not generated")
	}
	os.Remove(rotateName)
	os.Remove("test3.log")
}

func TestFileRotate_02(t *testing.T) {
	fn1 := "rotate_day.log"
	fn2 := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".log"
	testFileRotate(t, fn1, fn2)
}

func TestFileRotate_03(t *testing.T) {
	fn1 := "rotate_day.log"
	fn := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".log"
	os.Create(fn)
	fn2 := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".001.log"
	testFileRotate(t, fn1, fn2)
	os.Remove(fn)
}

func TestFileRotate_04(t *testing.T) {
	fn1 := "rotate_day.log"
	fn2 := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".log"
	testFileDailyRotate(t, fn1, fn2)
}

func TestFileRotate_05(t *testing.T) {
	fn1 := "rotate_day.log"
	fn := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".log"
	os.Create(fn)
	fn2 := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".001.log"
	testFileDailyRotate(t, fn1, fn2)
	os.Remove(fn)
}

func testFileRotate(t *testing.T, fn1, fn2 string) {
	fw := &fileLogWriter{
		Daily:   true,
		MaxDays: 7,
		Rotate:  true,
		Level:   LevelTrace,
		Perm:    "0660",
	}
	fw.Init(fmt.Sprintf(`{"filename":"%v","maxdays":1}`, fn1))
	fw.dailyOpenTime = time.Now().Add(-24 * time.Hour)
	fw.dailyOpenDate = fw.dailyOpenTime.Day()
	fw.WriteMsg(time.Now(), "this is a msg for test", LevelDebug)

	for _, file := range []string{fn1, fn2} {
		_, err := os.Stat(file)
		if err != nil {
			t.FailNow()
		}
		os.Remove(file)
	}
	fw.Destroy()
}

func testFileDailyRotate(t *testing.T, fn1, fn2 string) {
	fw := &fileLogWriter{
		Daily:   true,
		MaxDays: 7,
		Rotate:  true,
		Level:   LevelTrace,
		Perm:    "0660",
	}
	fw.Init(fmt.Sprintf(`{"filename":"%v","maxdays":1}`, fn1))
	fw.dailyOpenTime = time.Now().Add(-24 * time.Hour)
	fw.dailyOpenDate = fw.dailyOpenTime.Day()
	today, _ := time.ParseInLocation("2006-01-02", time.Now().Format("2006-01-02"), fw.dailyOpenTime.Location())
	today = today.Add(-1 * time.Second)
	fw.dailyRotate(today)
	for _, file := range []string{fn1, fn2} {
		_, err := os.Stat(file)
		if err != nil {
			t.FailNow()
		}
		content, err := ioutil.ReadFile(file)
		if err != nil {
			t.FailNow()
		}
		if len(content) > 0 {
			t.FailNow()
		}
		os.Remove(file)
	}
	fw.Destroy()
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func BenchmarkFile(b *testing.B) {
	log := NewLogger(100000)
	log.SetLogger("file", `{"filename":"test4.log"}`)
	for i := 0; i < b.N; i++ {
		log.Debug("debug")
	}
	os.Remove("test4.log")
}

func BenchmarkFileAsynchronous(b *testing.B) {
	log := NewLogger(100000)
	log.SetLogger("file", `{"filename":"test4.log"}`)
	log.Async()
	for i := 0; i < b.N; i++ {
		log.Debug("debug")
	}
	os.Remove("test4.log")
}

func BenchmarkFileCallDepth(b *testing.B) {
	log := NewLogger(100000)
	log.SetLogger("file", `{"filename":"test4.log"}`)
	log.EnableFuncCallDepth(true)
	log.SetLogFuncCallDepth(2)
	for i := 0; i < b.N; i++ {
		log.Debug("debug")
	}
	os.Remove("test4.log")
}

func BenchmarkFileAsynchronousCallDepth(b *testing.B) {
	log := NewLogger(100000)
	log.SetLogger("file", `{"filename":"test4.log"}`)
	log.EnableFuncCallDepth(true)
	log.SetLogFuncCallDepth(2)
	log.Async()
	for i := 0; i < b.N; i++ {
		log.Debug("debug")
	}
	os.Remove("test4.log")
}

func BenchmarkFileOnGoroutine(b *testing.B) {
	log := NewLogger(100000)
	log.SetLogger("file", `{"filename":"test4.log"}`)
	for i := 0; i < b.N; i++ {
		go log.Debug("debug")
	}
	os.Remove("test4.log")
}
