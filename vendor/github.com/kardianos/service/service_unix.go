// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

// +build linux darwin

package service

import (
	"fmt"
	"io"
	"io/ioutil"
	"log/syslog"
	"os/exec"
	"syscall"
)

func newSysLogger(name string, errs chan<- error) (Logger, error) {
	w, err := syslog.New(syslog.LOG_INFO, name)
	if err != nil {
		return nil, err
	}
	return sysLogger{w, errs}, nil
}

type sysLogger struct {
	*syslog.Writer
	errs chan<- error
}

func (s sysLogger) send(err error) error {
	if err != nil && s.errs != nil {
		s.errs <- err
	}
	return err
}

func (s sysLogger) Error(v ...interface{}) error {
	return s.send(s.Writer.Err(fmt.Sprint(v...)))
}
func (s sysLogger) Warning(v ...interface{}) error {
	return s.send(s.Writer.Warning(fmt.Sprint(v...)))
}
func (s sysLogger) Info(v ...interface{}) error {
	return s.send(s.Writer.Info(fmt.Sprint(v...)))
}
func (s sysLogger) Errorf(format string, a ...interface{}) error {
	return s.send(s.Writer.Err(fmt.Sprintf(format, a...)))
}
func (s sysLogger) Warningf(format string, a ...interface{}) error {
	return s.send(s.Writer.Warning(fmt.Sprintf(format, a...)))
}
func (s sysLogger) Infof(format string, a ...interface{}) error {
	return s.send(s.Writer.Info(fmt.Sprintf(format, a...)))
}

func run(command string, arguments ...string) error {
	_, _, err := runCommand(command, false, arguments...)
	return err
}

func runWithOutput(command string, arguments ...string) (int, string, error) {
	return runCommand(command, true, arguments...)
}

func runCommand(command string, readStdout bool, arguments ...string) (int, string, error) {
	cmd := exec.Command(command, arguments...)

	var output string
	var stdout io.ReadCloser
	var err error

	if readStdout {
		// Connect pipe to read Stdout
		stdout, err = cmd.StdoutPipe()

		if err != nil {
			// Failed to connect pipe
			return 0, "", fmt.Errorf("%q failed to connect stdout pipe: %v", command, err)
		}
	}

	// Connect pipe to read Stderr
	stderr, err := cmd.StderrPipe()

	if err != nil {
		// Failed to connect pipe
		return 0, "", fmt.Errorf("%q failed to connect stderr pipe: %v", command, err)
	}

	// Do not use cmd.Run()
	if err := cmd.Start(); err != nil {
		// Problem while copying stdin, stdout, or stderr
		return 0, "", fmt.Errorf("%q failed: %v", command, err)
	}

	// Zero exit status
	// Darwin: launchctl can fail with a zero exit status,
	// so check for emtpy stderr
	if command == "launchctl" {
		slurp, _ := ioutil.ReadAll(stderr)
		if len(slurp) > 0 {
			return 0, "", fmt.Errorf("%q failed with stderr: %s", command, slurp)
		}
	}

	if readStdout {
		out, err := ioutil.ReadAll(stdout)
		if err != nil {
			return 0, "", fmt.Errorf("%q failed while attempting to read stdout: %v", command, err)
		} else if len(out) > 0 {
			output = string(out)
		}
	}

	if err := cmd.Wait(); err != nil {
		exitStatus, ok := isExitError(err)
		if ok {
			// Command didn't exit with a zero exit status.
			return exitStatus, output, err
		}

		// An error occurred and there is no exit status.
		return 0, output, fmt.Errorf("%q failed: %v", command, err)
	}

	return 0, output, nil
}

func isExitError(err error) (int, bool) {
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), true
		}
	}

	return 0, false
}
