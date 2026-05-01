// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !frps

package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

const (
	// mcpStdioReapInterval is how often the worker checks for an idle child.
	mcpStdioReapInterval = 30 * time.Second
	// mcpStdioReadTimeout is the per-request deadline for a child response.
	mcpStdioReadTimeout = 30 * time.Second
)

func init() {
	Register(v1.PluginMCPStdio, NewMCPStdioPlugin)
}

// MCPStdioPlugin exposes a stdio-based MCP server (a child process that speaks
// newline-delimited JSON-RPC on stdin/stdout) over the Streamable HTTP
// transport. Each inbound HTTP POST body is forwarded as one JSON-RPC line;
// the next line read from the child's stdout is returned as the HTTP response.
//
// The child is spawned lazily on the first request and is killed when idle
// (when IdleTimeoutSeconds > 0). To keep MCP sessions intact across respawns,
// the plugin caches the most recent "initialize" request and the
// "notifications/initialized" notification, replaying them whenever a new
// child is started.
//
// A single worker goroutine owns the child process and all its I/O, so no
// mutexes are needed. HTTP handlers communicate with the worker via channels.
type MCPStdioPlugin struct {
	opts    *v1.MCPStdioPluginOptions
	idleTTL time.Duration

	l *Listener
	s *http.Server

	reqCh      chan dispatchReq
	closeCh    chan struct{}
	workerDone chan struct{}
	killCh     chan chan struct{} // signals worker to kill the child and ack
}

type dispatchReq struct {
	body    []byte
	replyCh chan dispatchResp
}

type dispatchResp struct {
	data []byte
	err  error
}

// NewMCPStdioPlugin constructs the plugin from validated options.
func NewMCPStdioPlugin(_ PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.MCPStdioPluginOptions)
	if len(opts.Command) == 0 {
		return nil, errors.New("mcp_stdio: command is required")
	}

	listener := NewProxyListener()

	p := &MCPStdioPlugin{
		opts:       opts,
		idleTTL:    time.Duration(opts.IdleTimeoutSeconds) * time.Second,
		l:          listener,
		reqCh:      make(chan dispatchReq),
		closeCh:    make(chan struct{}),
		workerDone: make(chan struct{}),
		killCh:     make(chan chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handle)
	p.s = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() { _ = p.s.Serve(listener) }()
	go p.worker()
	return p, nil
}

func (p *MCPStdioPlugin) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	body = bytes.TrimRight(body, "\r\n")
	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	resp, err := p.dispatch(r.Context(), body)
	if err != nil {
		log.Warnf("mcp_stdio: dispatch error: %v", err)
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if resp == nil {
		// Notifications carry no id and produce no response on stdio either;
		// per MCP HTTP spec we return 202 Accepted with an empty body.
		w.WriteHeader(http.StatusAccepted)
		return
	}
	_, _ = w.Write(resp)
}

// dispatch sends a JSON-RPC frame to the worker and waits for the response.
// It returns nil data for notifications (no id field), respecting ctx for
// cancellation throughout.
func (p *MCPStdioPlugin) dispatch(ctx context.Context, body []byte) ([]byte, error) {
	replyCh := make(chan dispatchResp, 1)
	select {
	case p.reqCh <- dispatchReq{body: body, replyCh: replyCh}:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.closeCh:
		return nil, errors.New("plugin closing")
	}
	select {
	case resp := <-replyCh:
		return resp.data, resp.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.closeCh:
		return nil, errors.New("plugin closing")
	}
}

// worker is the single goroutine that owns the child process and all its I/O.
// It also handles idle reaping. Because all child state lives here, no mutexes
// are needed.
func (p *MCPStdioPlugin) worker() {
	defer close(p.workerDone)

	var (
		child          *exec.Cmd
		childIn        io.WriteCloser
		childOut       *bufio.Reader
		lastUsedAt     time.Time
		cachedInitReq  []byte
		cachedInitNote []byte
	)

	killChild := func() {
		if child == nil {
			return
		}
		if childIn != nil {
			_ = childIn.Close()
		}
		if child.Process != nil {
			_ = child.Process.Kill()
		}
		_ = child.Wait()
		child = nil
		childIn = nil
		childOut = nil
	}

	ensureChild := func() error {
		if child != nil {
			return nil
		}
		cmd := exec.Command(p.opts.Command[0], p.opts.Command[1:]...)
		cmd.Env = os.Environ()
		for k, v := range p.opts.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		go pipeStderrToLog(stderr, p.opts.Command[0])

		child = cmd
		childIn = stdin
		childOut = bufio.NewReader(stdout)
		lastUsedAt = time.Now()

		if cachedInitReq != nil {
			if _, err := fmt.Fprintf(childIn, "%s\n", cachedInitReq); err != nil {
				killChild()
				return fmt.Errorf("replay initialize: %w", err)
			}
			if _, err := childOut.ReadBytes('\n'); err != nil {
				killChild()
				return fmt.Errorf("replay initialize read: %w", err)
			}
		}
		if cachedInitNote != nil {
			if _, err := fmt.Fprintf(childIn, "%s\n", cachedInitNote); err != nil {
				killChild()
				return fmt.Errorf("replay initialized notification: %w", err)
			}
		}
		return nil
	}

	var reaperCh <-chan time.Time
	if p.idleTTL > 0 {
		t := time.NewTicker(mcpStdioReapInterval)
		defer t.Stop()
		reaperCh = t.C
	}

	for {
		select {
		case <-p.closeCh:
			killChild()
			return

		case ack := <-p.killCh:
			killChild()
			close(ack)

		case <-reaperCh:
			if child != nil && time.Since(lastUsedAt) > p.idleTTL {
				log.Infof("mcp_stdio: reaping idle child %s", p.opts.Command[0])
				killChild()
			}

		case req := <-p.reqCh:
			hasID, isInit, isInitNote := classifyJSONRPC(req.body)

			if err := ensureChild(); err != nil {
				req.replyCh <- dispatchResp{err: fmt.Errorf("spawn child: %w", err)}
				continue
			}

			if isInit {
				cachedInitReq = append(cachedInitReq[:0], req.body...)
			}
			if isInitNote {
				cachedInitNote = append(cachedInitNote[:0], req.body...)
			}

			if _, err := fmt.Fprintf(childIn, "%s\n", req.body); err != nil {
				killChild()
				req.replyCh <- dispatchResp{err: fmt.Errorf("write stdin: %w", err)}
				continue
			}
			lastUsedAt = time.Now()

			if !hasID {
				req.replyCh <- dispatchResp{}
				continue
			}

			// Read the child's response in a goroutine so we can enforce a
			// timeout and still react to plugin shutdown. The channel is
			// buffered so the goroutine never leaks: if we time out and kill
			// the child, the closed pipe causes ReadBytes to return an error
			// and the goroutine exits normally.
			type readResult struct {
				line []byte
				err  error
			}
			readCh := make(chan readResult, 1)
			go func() {
				line, err := childOut.ReadBytes('\n')
				readCh <- readResult{line, err}
			}()

			select {
			case r := <-readCh:
				if r.err != nil {
					killChild()
					req.replyCh <- dispatchResp{err: fmt.Errorf("read stdout: %w", r.err)}
				} else {
					lastUsedAt = time.Now()
					req.replyCh <- dispatchResp{data: bytes.TrimRight(r.line, "\r\n")}
				}
			case <-time.After(mcpStdioReadTimeout):
				killChild()
				req.replyCh <- dispatchResp{err: errors.New("child response timeout")}
			case <-p.closeCh:
				killChild()
				req.replyCh <- dispatchResp{err: errors.New("plugin closing")}
				return
			}
		}
	}
}

// classifyJSONRPC inspects a JSON-RPC frame's "method" and "id" fields.
// hasID is true when the frame has a non-null id (and therefore expects a
// response); isInit/isInitNote flag the MCP handshake messages so they can
// be cached for replay.
func classifyJSONRPC(body []byte) (hasID, isInit, isInitNote bool) {
	var head struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(body, &head); err != nil {
		// Be conservative: assume the peer expects a response so we surface
		// any framing error back to the client instead of hanging.
		return true, false, false
	}
	hasID = len(head.ID) > 0 && string(head.ID) != "null"
	isInit = head.Method == "initialize"
	isInitNote = head.Method == "notifications/initialized"
	return
}

func pipeStderrToLog(r io.Reader, name string) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			log.Infof("mcp_stdio[%s]: %s", name, bytes.TrimRight([]byte(line), "\r\n"))
		}
		if err != nil {
			return
		}
	}
}

// killChildNow asks the worker to kill the current child process and waits for
// it to complete. Useful in tests to simulate an idle reap without waiting for
// the TTL.
func (p *MCPStdioPlugin) killChildNow() {
	ack := make(chan struct{})
	select {
	case p.killCh <- ack:
		<-ack
	case <-p.closeCh:
	}
}

func (p *MCPStdioPlugin) Handle(_ context.Context, connInfo *ConnectionInfo) {
	wrapConn := netpkg.WrapReadWriteCloserToConn(connInfo.Conn, connInfo.UnderlyingConn)
	_ = p.l.PutConn(wrapConn)
}

func (p *MCPStdioPlugin) Name() string {
	return v1.PluginMCPStdio
}

func (p *MCPStdioPlugin) Close() error {
	_ = p.s.Close()
	_ = p.l.Close()
	close(p.closeCh)
	<-p.workerDone
	return nil
}
