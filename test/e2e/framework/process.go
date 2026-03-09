package framework

import (
	"fmt"
	"maps"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	flog "github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/process"
)

// RunProcesses starts one frps and zero or more frpc processes from templates.
func (f *Framework) RunProcesses(serverTemplate string, clientTemplates []string) (*process.Process, []*process.Process) {
	templates := append([]string{serverTemplate}, clientTemplates...)
	outs, ports, err := f.RenderTemplates(templates)
	ExpectNoError(err)

	maps.Copy(f.usedPorts, ports)

	// Start frps.
	serverPath := filepath.Join(f.TempDirectory, "frp-e2e-server-0")
	err = os.WriteFile(serverPath, []byte(outs[0]), 0o600)
	ExpectNoError(err)

	if TestContext.Debug {
		flog.Debugf("[%s] %s", serverPath, outs[0])
	}

	serverProcess := process.NewWithEnvs(TestContext.FRPServerPath, []string{"-c", serverPath}, f.osEnvs)
	f.serverConfPaths = append(f.serverConfPaths, serverPath)
	f.serverProcesses = append(f.serverProcesses, serverProcess)
	err = serverProcess.Start()
	ExpectNoError(err)

	if port, ok := ports[consts.PortServerName]; ok {
		ExpectNoError(WaitForTCPReady(net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 5*time.Second))
	} else {
		time.Sleep(2 * time.Second)
	}

	// Start frpc(s).
	clientProcesses := make([]*process.Process, 0, len(clientTemplates))
	for i := range clientTemplates {
		path := filepath.Join(f.TempDirectory, fmt.Sprintf("frp-e2e-client-%d", i))
		err = os.WriteFile(path, []byte(outs[1+i]), 0o600)
		ExpectNoError(err)

		if TestContext.Debug {
			flog.Debugf("[%s] %s", path, outs[1+i])
		}

		p := process.NewWithEnvs(TestContext.FRPClientPath, []string{"-c", path}, f.osEnvs)
		f.clientConfPaths = append(f.clientConfPaths, path)
		f.clientProcesses = append(f.clientProcesses, p)
		clientProcesses = append(clientProcesses, p)
		err = p.Start()
		ExpectNoError(err)
	}
	// frpc needs time to connect and register proxies with frps.
	if len(clientProcesses) > 0 {
		time.Sleep(1500 * time.Millisecond)
	}

	return serverProcess, clientProcesses
}

func (f *Framework) RunFrps(args ...string) (*process.Process, string, error) {
	p := process.NewWithEnvs(TestContext.FRPServerPath, args, f.osEnvs)
	f.serverProcesses = append(f.serverProcesses, p)
	err := p.Start()
	if err != nil {
		return p, p.Output(), err
	}
	select {
	case <-p.Done():
	case <-time.After(2 * time.Second):
	}
	return p, p.Output(), nil
}

func (f *Framework) RunFrpc(args ...string) (*process.Process, string, error) {
	p := process.NewWithEnvs(TestContext.FRPClientPath, args, f.osEnvs)
	f.clientProcesses = append(f.clientProcesses, p)
	err := p.Start()
	if err != nil {
		return p, p.Output(), err
	}
	select {
	case <-p.Done():
	case <-time.After(1500 * time.Millisecond):
	}
	return p, p.Output(), nil
}

func (f *Framework) GenerateConfigFile(content string) string {
	f.configFileIndex++
	path := filepath.Join(f.TempDirectory, fmt.Sprintf("frp-e2e-config-%d", f.configFileIndex))
	err := os.WriteFile(path, []byte(content), 0o600)
	ExpectNoError(err)
	return path
}

// WaitForTCPReady polls a TCP address until a connection succeeds or timeout.
func WaitForTCPReady(addr string, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("invalid timeout for TCP readiness on %s: timeout must be positive", addr)
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	if lastErr == nil {
		return fmt.Errorf("timeout waiting for TCP readiness on %s before any dial attempt", addr)
	}
	return fmt.Errorf("timeout waiting for TCP readiness on %s: %w", addr, lastErr)
}
