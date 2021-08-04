// Package ginkgowrapper wraps Ginkgo Fail and Skip functions to panic
// with structured data instead of a constant string.
package ginkgowrapper

import (
	"bufio"
	"bytes"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/onsi/ginkgo"
)

// FailurePanic is the value that will be panicked from Fail.
type FailurePanic struct {
	Message        string // The failure message passed to Fail
	Filename       string // The filename that is the source of the failure
	Line           int    // The line number of the filename that is the source of the failure
	FullStackTrace string // A full stack trace starting at the source of the failure
}

// String makes FailurePanic look like the old Ginkgo panic when printed.
func (FailurePanic) String() string { return ginkgo.GINKGO_PANIC }

// Fail wraps ginkgo.Fail so that it panics with more useful
// information about the failure. This function will panic with a
// FailurePanic.
func Fail(message string, callerSkip ...int) {
	skip := 1
	if len(callerSkip) > 0 {
		skip += callerSkip[0]
	}

	_, file, line, _ := runtime.Caller(skip)
	fp := FailurePanic{
		Message:        message,
		Filename:       file,
		Line:           line,
		FullStackTrace: pruneStack(skip),
	}

	defer func() {
		e := recover()
		if e != nil {
			panic(fp)
		}
	}()

	ginkgo.Fail(message, skip)
}

// ginkgo adds a lot of test running infrastructure to the stack, so
// we filter those out
var stackSkipPattern = regexp.MustCompile(`onsi/ginkgo`)

func pruneStack(skip int) string {
	skip += 2 // one for pruneStack and one for debug.Stack
	stack := debug.Stack()
	scanner := bufio.NewScanner(bytes.NewBuffer(stack))
	var prunedStack []string

	// skip the top of the stack
	for i := 0; i < 2*skip+1; i++ {
		scanner.Scan()
	}

	for scanner.Scan() {
		if stackSkipPattern.Match(scanner.Bytes()) {
			scanner.Scan() // these come in pairs
		} else {
			prunedStack = append(prunedStack, scanner.Text())
			scanner.Scan() // these come in pairs
			prunedStack = append(prunedStack, scanner.Text())
		}
	}

	return strings.Join(prunedStack, "\n")
}
