package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SafeBuffer is a thread-safe wrapper around bytes.Buffer.
// It is safe to call Write and String concurrently.
type SafeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *SafeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *SafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type Process struct {
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	errorOutput *SafeBuffer
	stdOutput   *SafeBuffer

	done     chan struct{}
	closeOne sync.Once
	waitErr  error

	started           bool
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
	p.errorOutput = &SafeBuffer{}
	p.stdOutput = &SafeBuffer{}
	cmd.Stderr = p.errorOutput
	cmd.Stdout = p.stdOutput
	return p
}

func (p *Process) Start() error {
	if p.started {
		return errors.New("process already started")
	}
	p.started = true

	err := p.cmd.Start()
	if err != nil {
		p.waitErr = err
		p.closeDone()
		return err
	}
	go func() {
		p.waitErr = p.cmd.Wait()
		p.closeDone()
	}()
	return nil
}

func (p *Process) closeDone() {
	p.closeOne.Do(func() { close(p.done) })
}

// Done returns a channel that is closed when the process exits.
func (p *Process) Done() <-chan struct{} {
	return p.done
}

func (p *Process) Stop() error {
	if p.stopped || !p.started {
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

// CountOutput returns how many times pattern appears in the current accumulated output.
func (p *Process) CountOutput(pattern string) int {
	return strings.Count(p.Output(), pattern)
}

func (p *Process) SetBeforeStopHandler(fn func()) {
	p.beforeStopHandler = fn
}

// WaitForOutput polls the combined process output until the pattern is found
// count time(s) or the timeout is reached. It also returns early if the process exits.
func (p *Process) WaitForOutput(pattern string, count int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		output := p.Output()
		if strings.Count(output, pattern) >= count {
			return nil
		}
		select {
		case <-p.Done():
			// Process exited, check one last time.
			output = p.Output()
			if strings.Count(output, pattern) >= count {
				return nil
			}
			return fmt.Errorf("process exited before %d occurrence(s) of %q found", count, pattern)
		case <-time.After(25 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for %d occurrence(s) of %q", count, pattern)
}
