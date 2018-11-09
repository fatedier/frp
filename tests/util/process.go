package util

import (
	"context"
	"os/exec"
)

type Process struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func NewProcess(path string, params []string) *Process {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, path, params...)
	return &Process{
		cmd:    cmd,
		cancel: cancel,
	}
}

func (p *Process) Start() error {
	return p.cmd.Start()
}

func (p *Process) Stop() error {
	p.cancel()
	return p.cmd.Wait()
}
