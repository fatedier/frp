// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Server-Sent Events
// W3C Working Draft 29 October 2009
// http://www.w3.org/TR/2009/WD-eventsource-20091029/

const ContentType = "text/event-stream"

var contentType = []string{ContentType}
var noCache = []string{"no-cache"}

var fieldReplacer = strings.NewReplacer(
	"\n", "\\n",
	"\r", "\\r")

var dataReplacer = strings.NewReplacer(
	"\n", "\ndata:",
	"\r", "\\r")

type Event struct {
	Event string
	Id    string
	Retry uint
	Data  interface{}
}

func Encode(writer io.Writer, event Event) error {
	w := checkWriter(writer)
	writeId(w, event.Id)
	writeEvent(w, event.Event)
	writeRetry(w, event.Retry)
	return writeData(w, event.Data)
}

func writeId(w stringWriter, id string) {
	if len(id) > 0 {
		w.WriteString("id:")
		fieldReplacer.WriteString(w, id)
		w.WriteString("\n")
	}
}

func writeEvent(w stringWriter, event string) {
	if len(event) > 0 {
		w.WriteString("event:")
		fieldReplacer.WriteString(w, event)
		w.WriteString("\n")
	}
}

func writeRetry(w stringWriter, retry uint) {
	if retry > 0 {
		w.WriteString("retry:")
		w.WriteString(strconv.FormatUint(uint64(retry), 10))
		w.WriteString("\n")
	}
}

func writeData(w stringWriter, data interface{}) error {
	w.WriteString("data:")
	switch kindOfData(data) {
	case reflect.Struct, reflect.Slice, reflect.Map:
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			return err
		}
		w.WriteString("\n")
	default:
		dataReplacer.WriteString(w, fmt.Sprint(data))
		w.WriteString("\n\n")
	}
	return nil
}

func (r Event) Render(w http.ResponseWriter) error {
	header := w.Header()
	header["Content-Type"] = contentType

	if _, exist := header["Cache-Control"]; !exist {
		header["Cache-Control"] = noCache
	}
	return Encode(w, r)
}

func kindOfData(data interface{}) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()
	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}
