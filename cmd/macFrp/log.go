package main

/*
#include <stdio.h>
typedef void (*LogListener)(const char* log);

LogListener logListener;

void setLogListener(LogListener l) {
	logListener = l;
}

void callback(const char* log) {
	if (logListener) {
        logListener(log);
	}
}

void cListener(const char* log) {
	printf("%s", log);
}

void testLog() {
	setLogListener(cListener);
}
*/
import "C"
import (
	"time"

	"github.com/fatedier/frp/pkg/util/log"
)

var l logForMacListener

type logForMacListener struct {
	log.LogListener
}

func (l *logForMacListener) Log(log string) {
	C.callback(C.CString(log))
}
func (l *logForMacListener) Location() string {
	location, _ := time.LoadLocation("Local")
	return location.String()
}

func init() {
	l = logForMacListener{}
	log.AppendListener(&l)
}

func logLog() {
	C.testLog()
	println(C.logListener)
}
