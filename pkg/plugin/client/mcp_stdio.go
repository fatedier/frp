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
	"sync"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

// mcpStdioReapInterval is how often the idle reaper checks the child process.
// It is a constant rather than a config option to keep the plugin surface small.
const mcpStdioReapInterval = 30 * time.Second

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
type MCPStdioPlugin struct {
	opts    *v1.MCPStdioPluginOptions
	idleTTL time.Duration

	l *Listener
	s *http.Server

	reaperCancel context.CancelFunc
	reaperDone   chan struct{}

	mu             sync.Mutex
	child          *exec.Cmd
	childIn        io.WriteCloser
	childOut       *bufio.Reader
	lastUsedAt     time.Time
	cachedInitReq  []byte
	cachedInitNote []byte
}

// NewMCPStdioPlugin constructs the plugin from validated options.
func NewMCPStdioPlugin(_ PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.MCPStdioPluginOptions)
	if len(opts.Command) == 0 {
		return nil, errors.New("mcp_stdio: command is required")
	}

	idle := time.Duration(opts.IdleTimeoutSeconds) * time.Second

	listener := NewProxyListener()
	reaperCtx, reaperCancel := context.WithCancel(context.Background())

	p := &MCPStdioPlugin{
		opts:         opts,
		idleTTL:      idle,
		l:            listener,
		reaperCancel: reaperCancel,
		reaperDone:   make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handle)
	p.s = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() { _ = p.s.Serve(listener) }()
	if idle > 0 {
		go p.reapLoop(reaperCtx)
	} else {
		close(p.reaperDone)
	}
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

	resp, err := p.dispatch(body)
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

// dispatch forwards a single JSON-RPC frame to the child and returns the
// response line. A nil response means the frame was a notification (no id)
// and no response is expected.
func (p *MCPStdioPlugin) dispatch(body []byte) ([]byte, error) {
	hasID, isInit, isInitNote := classifyJSONRPC(body)

	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureChildLocked(); err != nil {
		return nil, fmt.Errorf("spawn child: %w", err)
	}

	if isInit {
		p.cachedInitReq = append(p.cachedInitReq[:0], body...)
	}
	if isInitNote {
		p.cachedInitNote = append(p.cachedInitNote[:0], body...)
	}

	if _, err := fmt.Fprintf(p.childIn, "%s\n", body); err != nil {
		p.killChildLocked()
		return nil, fmt.Errorf("write stdin: %w", err)
	}
	p.lastUsedAt = time.Now()

	if !hasID {
		return nil, nil
	}

	line, err := p.childOut.ReadBytes('\n')
	if err != nil {
		p.killChildLocked()
		return nil, fmt.Errorf("read stdout: %w", err)
	}
	p.lastUsedAt = time.Now()
	return bytes.TrimRight(line, "\r\n"), nil
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

// ensureChildLocked starts the child if not running, replaying the cached
// MCP handshake so a new process appears initialized to subsequent callers.
// mu must be held.
func (p *MCPStdioPlugin) ensureChildLocked() error {
	if p.child != nil {
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

	p.child = cmd
	p.childIn = stdin
	p.childOut = bufio.NewReader(stdout)
	p.lastUsedAt = time.Now()

	if p.cachedInitReq != nil {
		if _, err := fmt.Fprintf(p.childIn, "%s\n", p.cachedInitReq); err != nil {
			p.killChildLocked()
			return fmt.Errorf("replay initialize: %w", err)
		}
		if _, err := p.childOut.ReadBytes('\n'); err != nil {
			p.killChildLocked()
			return fmt.Errorf("replay initialize read: %w", err)
		}
	}
	if p.cachedInitNote != nil {
		if _, err := fmt.Fprintf(p.childIn, "%s\n", p.cachedInitNote); err != nil {
			p.killChildLocked()
			return fmt.Errorf("replay initialized notification: %w", err)
		}
	}
	return nil
}

func (p *MCPStdioPlugin) killChildLocked() {
	if p.child == nil {
		return
	}
	if p.childIn != nil {
		_ = p.childIn.Close()
	}
	if proc := p.child.Process; proc != nil {
		_ = proc.Kill()
	}
	_ = p.child.Wait()
	p.child = nil
	p.childIn = nil
	p.childOut = nil
}

func (p *MCPStdioPlugin) reapLoop(ctx context.Context) {
	defer close(p.reaperDone)
	t := time.NewTicker(mcpStdioReapInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.mu.Lock()
			if p.child != nil && time.Since(p.lastUsedAt) > p.idleTTL {
				log.Infof("mcp_stdio: reaping idle child %s", p.opts.Command[0])
				p.killChildLocked()
			}
			p.mu.Unlock()
		}
	}
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

func (p *MCPStdioPlugin) Handle(_ context.Context, connInfo *ConnectionInfo) {
	wrapConn := netpkg.WrapReadWriteCloserToConn(connInfo.Conn, connInfo.UnderlyingConn)
	_ = p.l.PutConn(wrapConn)
}

func (p *MCPStdioPlugin) Name() string {
	return v1.PluginMCPStdio
}

func (p *MCPStdioPlugin) Close() error {
	p.reaperCancel()
	<-p.reaperDone
	_ = p.s.Close()
	_ = p.l.Close()
	p.mu.Lock()
	p.killChildLocked()
	p.mu.Unlock()
	return nil
}
