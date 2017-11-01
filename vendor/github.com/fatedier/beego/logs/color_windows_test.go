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

// +build windows

package logs

import (
	"bytes"
	"fmt"
	"syscall"
	"testing"
)

var GetConsoleScreenBufferInfo = getConsoleScreenBufferInfo

func ChangeColor(color uint16) {
	setConsoleTextAttribute(uintptr(syscall.Stdout), color)
}

func ResetColor() {
	ChangeColor(uint16(0x0007))
}

func TestWritePlanText(t *testing.T) {
	inner := bytes.NewBufferString("")
	w := NewAnsiColorWriter(inner)
	expected := "plain text"
	fmt.Fprintf(w, expected)
	actual := inner.String()
	if actual != expected {
		t.Errorf("Get %q, want %q", actual, expected)
	}
}

func TestWriteParseText(t *testing.T) {
	inner := bytes.NewBufferString("")
	w := NewAnsiColorWriter(inner)

	inputTail := "\x1b[0mtail text"
	expectedTail := "tail text"
	fmt.Fprintf(w, inputTail)
	actualTail := inner.String()
	inner.Reset()
	if actualTail != expectedTail {
		t.Errorf("Get %q, want %q", actualTail, expectedTail)
	}

	inputHead := "head text\x1b[0m"
	expectedHead := "head text"
	fmt.Fprintf(w, inputHead)
	actualHead := inner.String()
	inner.Reset()
	if actualHead != expectedHead {
		t.Errorf("Get %q, want %q", actualHead, expectedHead)
	}

	inputBothEnds := "both ends \x1b[0m text"
	expectedBothEnds := "both ends  text"
	fmt.Fprintf(w, inputBothEnds)
	actualBothEnds := inner.String()
	inner.Reset()
	if actualBothEnds != expectedBothEnds {
		t.Errorf("Get %q, want %q", actualBothEnds, expectedBothEnds)
	}

	inputManyEsc := "\x1b\x1b\x1b\x1b[0m many esc"
	expectedManyEsc := "\x1b\x1b\x1b many esc"
	fmt.Fprintf(w, inputManyEsc)
	actualManyEsc := inner.String()
	inner.Reset()
	if actualManyEsc != expectedManyEsc {
		t.Errorf("Get %q, want %q", actualManyEsc, expectedManyEsc)
	}

	expectedSplit := "split  text"
	for _, ch := range "split \x1b[0m text" {
		fmt.Fprintf(w, string(ch))
	}
	actualSplit := inner.String()
	inner.Reset()
	if actualSplit != expectedSplit {
		t.Errorf("Get %q, want %q", actualSplit, expectedSplit)
	}
}

type screenNotFoundError struct {
	error
}

func writeAnsiColor(expectedText, colorCode string) (actualText string, actualAttributes uint16, err error) {
	inner := bytes.NewBufferString("")
	w := NewAnsiColorWriter(inner)
	fmt.Fprintf(w, "\x1b[%sm%s", colorCode, expectedText)

	actualText = inner.String()
	screenInfo := GetConsoleScreenBufferInfo(uintptr(syscall.Stdout))
	if screenInfo != nil {
		actualAttributes = screenInfo.WAttributes
	} else {
		err = &screenNotFoundError{}
	}
	return
}

type testParam struct {
	text       string
	attributes uint16
	ansiColor  string
}

func TestWriteAnsiColorText(t *testing.T) {
	screenInfo := GetConsoleScreenBufferInfo(uintptr(syscall.Stdout))
	if screenInfo == nil {
		t.Fatal("Could not get ConsoleScreenBufferInfo")
	}
	defer ChangeColor(screenInfo.WAttributes)
	defaultFgColor := screenInfo.WAttributes & uint16(0x0007)
	defaultBgColor := screenInfo.WAttributes & uint16(0x0070)
	defaultFgIntensity := screenInfo.WAttributes & uint16(0x0008)
	defaultBgIntensity := screenInfo.WAttributes & uint16(0x0080)

	fgParam := []testParam{
		{"foreground black  ", uint16(0x0000 | 0x0000), "30"},
		{"foreground red    ", uint16(0x0004 | 0x0000), "31"},
		{"foreground green  ", uint16(0x0002 | 0x0000), "32"},
		{"foreground yellow ", uint16(0x0006 | 0x0000), "33"},
		{"foreground blue   ", uint16(0x0001 | 0x0000), "34"},
		{"foreground magenta", uint16(0x0005 | 0x0000), "35"},
		{"foreground cyan   ", uint16(0x0003 | 0x0000), "36"},
		{"foreground white  ", uint16(0x0007 | 0x0000), "37"},
		{"foreground default", defaultFgColor | 0x0000, "39"},
		{"foreground light gray   ", uint16(0x0000 | 0x0008 | 0x0000), "90"},
		{"foreground light red    ", uint16(0x0004 | 0x0008 | 0x0000), "91"},
		{"foreground light green  ", uint16(0x0002 | 0x0008 | 0x0000), "92"},
		{"foreground light yellow ", uint16(0x0006 | 0x0008 | 0x0000), "93"},
		{"foreground light blue   ", uint16(0x0001 | 0x0008 | 0x0000), "94"},
		{"foreground light magenta", uint16(0x0005 | 0x0008 | 0x0000), "95"},
		{"foreground light cyan   ", uint16(0x0003 | 0x0008 | 0x0000), "96"},
		{"foreground light white  ", uint16(0x0007 | 0x0008 | 0x0000), "97"},
	}

	bgParam := []testParam{
		{"background black  ", uint16(0x0007 | 0x0000), "40"},
		{"background red    ", uint16(0x0007 | 0x0040), "41"},
		{"background green  ", uint16(0x0007 | 0x0020), "42"},
		{"background yellow ", uint16(0x0007 | 0x0060), "43"},
		{"background blue   ", uint16(0x0007 | 0x0010), "44"},
		{"background magenta", uint16(0x0007 | 0x0050), "45"},
		{"background cyan   ", uint16(0x0007 | 0x0030), "46"},
		{"background white  ", uint16(0x0007 | 0x0070), "47"},
		{"background default", uint16(0x0007) | defaultBgColor, "49"},
		{"background light gray   ", uint16(0x0007 | 0x0000 | 0x0080), "100"},
		{"background light red    ", uint16(0x0007 | 0x0040 | 0x0080), "101"},
		{"background light green  ", uint16(0x0007 | 0x0020 | 0x0080), "102"},
		{"background light yellow ", uint16(0x0007 | 0x0060 | 0x0080), "103"},
		{"background light blue   ", uint16(0x0007 | 0x0010 | 0x0080), "104"},
		{"background light magenta", uint16(0x0007 | 0x0050 | 0x0080), "105"},
		{"background light cyan   ", uint16(0x0007 | 0x0030 | 0x0080), "106"},
		{"background light white  ", uint16(0x0007 | 0x0070 | 0x0080), "107"},
	}

	resetParam := []testParam{
		{"all reset", defaultFgColor | defaultBgColor | defaultFgIntensity | defaultBgIntensity, "0"},
		{"all reset", defaultFgColor | defaultBgColor | defaultFgIntensity | defaultBgIntensity, ""},
	}

	boldParam := []testParam{
		{"bold on", uint16(0x0007 | 0x0008), "1"},
		{"bold off", uint16(0x0007), "21"},
	}

	underscoreParam := []testParam{
		{"underscore on", uint16(0x0007 | 0x8000), "4"},
		{"underscore off", uint16(0x0007), "24"},
	}

	blinkParam := []testParam{
		{"blink on", uint16(0x0007 | 0x0080), "5"},
		{"blink off", uint16(0x0007), "25"},
	}

	mixedParam := []testParam{
		{"both black,   bold, underline, blink", uint16(0x0000 | 0x0000 | 0x0008 | 0x8000 | 0x0080), "30;40;1;4;5"},
		{"both red,     bold, underline, blink", uint16(0x0004 | 0x0040 | 0x0008 | 0x8000 | 0x0080), "31;41;1;4;5"},
		{"both green,   bold, underline, blink", uint16(0x0002 | 0x0020 | 0x0008 | 0x8000 | 0x0080), "32;42;1;4;5"},
		{"both yellow,  bold, underline, blink", uint16(0x0006 | 0x0060 | 0x0008 | 0x8000 | 0x0080), "33;43;1;4;5"},
		{"both blue,    bold, underline, blink", uint16(0x0001 | 0x0010 | 0x0008 | 0x8000 | 0x0080), "34;44;1;4;5"},
		{"both magenta, bold, underline, blink", uint16(0x0005 | 0x0050 | 0x0008 | 0x8000 | 0x0080), "35;45;1;4;5"},
		{"both cyan,    bold, underline, blink", uint16(0x0003 | 0x0030 | 0x0008 | 0x8000 | 0x0080), "36;46;1;4;5"},
		{"both white,   bold, underline, blink", uint16(0x0007 | 0x0070 | 0x0008 | 0x8000 | 0x0080), "37;47;1;4;5"},
		{"both default, bold, underline, blink", uint16(defaultFgColor | defaultBgColor | 0x0008 | 0x8000 | 0x0080), "39;49;1;4;5"},
	}

	assertTextAttribute := func(expectedText string, expectedAttributes uint16, ansiColor string) {
		actualText, actualAttributes, err := writeAnsiColor(expectedText, ansiColor)
		if actualText != expectedText {
			t.Errorf("Get %q, want %q", actualText, expectedText)
		}
		if err != nil {
			t.Fatal("Could not get ConsoleScreenBufferInfo")
		}
		if actualAttributes != expectedAttributes {
			t.Errorf("Text: %q, Get 0x%04x, want 0x%04x", expectedText, actualAttributes, expectedAttributes)
		}
	}

	for _, v := range fgParam {
		ResetColor()
		assertTextAttribute(v.text, v.attributes, v.ansiColor)
	}

	for _, v := range bgParam {
		ChangeColor(uint16(0x0070 | 0x0007))
		assertTextAttribute(v.text, v.attributes, v.ansiColor)
	}

	for _, v := range resetParam {
		ChangeColor(uint16(0x0000 | 0x0070 | 0x0008))
		assertTextAttribute(v.text, v.attributes, v.ansiColor)
	}

	ResetColor()
	for _, v := range boldParam {
		assertTextAttribute(v.text, v.attributes, v.ansiColor)
	}

	ResetColor()
	for _, v := range underscoreParam {
		assertTextAttribute(v.text, v.attributes, v.ansiColor)
	}

	ResetColor()
	for _, v := range blinkParam {
		assertTextAttribute(v.text, v.attributes, v.ansiColor)
	}

	for _, v := range mixedParam {
		ResetColor()
		assertTextAttribute(v.text, v.attributes, v.ansiColor)
	}
}

func TestIgnoreUnknownSequences(t *testing.T) {
	inner := bytes.NewBufferString("")
	w := NewModeAnsiColorWriter(inner, OutputNonColorEscSeq)

	inputText := "\x1b[=decpath mode"
	expectedTail := inputText
	fmt.Fprintf(w, inputText)
	actualTail := inner.String()
	inner.Reset()
	if actualTail != expectedTail {
		t.Errorf("Get %q, want %q", actualTail, expectedTail)
	}

	inputText = "\x1b[=tailing esc and bracket\x1b["
	expectedTail = inputText
	fmt.Fprintf(w, inputText)
	actualTail = inner.String()
	inner.Reset()
	if actualTail != expectedTail {
		t.Errorf("Get %q, want %q", actualTail, expectedTail)
	}

	inputText = "\x1b[?tailing esc\x1b"
	expectedTail = inputText
	fmt.Fprintf(w, inputText)
	actualTail = inner.String()
	inner.Reset()
	if actualTail != expectedTail {
		t.Errorf("Get %q, want %q", actualTail, expectedTail)
	}

	inputText = "\x1b[1h;3punended color code invalid\x1b3"
	expectedTail = inputText
	fmt.Fprintf(w, inputText)
	actualTail = inner.String()
	inner.Reset()
	if actualTail != expectedTail {
		t.Errorf("Get %q, want %q", actualTail, expectedTail)
	}
}
