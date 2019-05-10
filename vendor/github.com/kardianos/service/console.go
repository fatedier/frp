// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

package service

import (
	"log"
	"os"
)

// ConsoleLogger logs to the std err.
var ConsoleLogger = consoleLogger{}

type consoleLogger struct {
	info, warn, err *log.Logger
}

func init() {
	ConsoleLogger.info = log.New(os.Stderr, "I: ", log.Ltime)
	ConsoleLogger.warn = log.New(os.Stderr, "W: ", log.Ltime)
	ConsoleLogger.err = log.New(os.Stderr, "E: ", log.Ltime)
}

func (c consoleLogger) Error(v ...interface{}) error {
	c.err.Print(v...)
	return nil
}
func (c consoleLogger) Warning(v ...interface{}) error {
	c.warn.Print(v...)
	return nil
}
func (c consoleLogger) Info(v ...interface{}) error {
	c.info.Print(v...)
	return nil
}
func (c consoleLogger) Errorf(format string, a ...interface{}) error {
	c.err.Printf(format, a...)
	return nil
}
func (c consoleLogger) Warningf(format string, a ...interface{}) error {
	c.warn.Printf(format, a...)
	return nil
}
func (c consoleLogger) Infof(format string, a ...interface{}) error {
	c.info.Printf(format, a...)
	return nil
}
