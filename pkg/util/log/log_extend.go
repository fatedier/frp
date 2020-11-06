package log

import (
	"sync"
	"time"
)

var logListeners = make([]LogListener, 0)
var lock sync.Mutex

type LogListener interface {
	Log(log string)
	Location() string
}

func AppendListener(l LogListener) {
	lock.Lock()
	logListeners = append(logListeners, l)
	lock.Unlock()
}

func CallLogListeners(log string) {
	lock.Lock()
	for _, l := range logListeners {
		location, _ := time.LoadLocation(l.Location())
		if location == nil {
			location = time.UTC
		}

		l.Log(time.Now().In(location).Format("2006/01/02 15:04:05") + ": " + log)
	}
	lock.Unlock()
}
