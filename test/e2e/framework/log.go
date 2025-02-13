package framework

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
)

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func log(level string, format string, args ...any) {
	fmt.Fprintf(ginkgo.GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

// Logf logs the info.
func Logf(format string, args ...any) {
	log("INFO", format, args...)
}

// Failf logs the fail info, including a stack trace starts with its direct caller
// (for example, for call chain f -> g -> Failf("foo", ...) error would be logged for "g").
func Failf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	skip := 1
	ginkgo.Fail(msg, skip)
	panic("unreachable")
}

// Fail is an alias for ginkgo.Fail.
var Fail = ginkgo.Fail
