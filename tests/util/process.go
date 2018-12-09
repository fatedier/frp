package util

import (
	"bytes"
	"context"
	"os/exec"
)

type Process struct {
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	errorOutput *bytes.Buffer

	beforeStopHandler func()
}

func NewProcess(path string, params []string) *Process {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, path, params...)
	p := &Process{
		cmd:    cmd,
		cancel: cancel,
	}
	p.errorOutput = bytes.NewBufferString("")
	cmd.Stderr = p.errorOutput
	return p
}

func (p *Process) Start() error {
	return p.cmd.Start()
}

func (p *Process) Stop() error {
	if p.beforeStopHandler != nil {
		p.beforeStopHandler()
	}
	p.cancel()
	return p.cmd.Wait()
}

func (p *Process) ErrorOutput() string {
	return p.errorOutput.String()
}

func (p *Process) SetBeforeStopHandler(fn func()) {
	p.beforeStopHandler = fn
}
