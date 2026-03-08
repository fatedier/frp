package process

import (
	"bytes"
	"context"
	"os/exec"
)

type Process struct {
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	errorOutput *bytes.Buffer
	stdOutput   *bytes.Buffer

	done    chan struct{}
	waitErr error

	beforeStopHandler func()
	stopped           bool
}

func New(path string, params []string) *Process {
	return NewWithEnvs(path, params, nil)
}

func NewWithEnvs(path string, params []string, envs []string) *Process {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, path, params...)
	cmd.Env = envs
	p := &Process{
		cmd:    cmd,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	p.errorOutput = bytes.NewBufferString("")
	p.stdOutput = bytes.NewBufferString("")
	cmd.Stderr = p.errorOutput
	cmd.Stdout = p.stdOutput
	return p
}

func (p *Process) Start() error {
	err := p.cmd.Start()
	if err != nil {
		p.waitErr = err
		close(p.done)
		return err
	}
	go func() {
		p.waitErr = p.cmd.Wait()
		close(p.done)
	}()
	return nil
}

// Done returns a channel that is closed when the process exits.
func (p *Process) Done() <-chan struct{} {
	return p.done
}

func (p *Process) Stop() error {
	if p.stopped {
		return nil
	}
	defer func() {
		p.stopped = true
	}()
	if p.beforeStopHandler != nil {
		p.beforeStopHandler()
	}
	p.cancel()
	<-p.done
	return p.waitErr
}

func (p *Process) ErrorOutput() string {
	return p.errorOutput.String()
}

func (p *Process) StdOutput() string {
	return p.stdOutput.String()
}

func (p *Process) Output() string {
	return p.stdOutput.String() + p.errorOutput.String()
}

func (p *Process) SetBeforeStopHandler(fn func()) {
	p.beforeStopHandler = fn
}
