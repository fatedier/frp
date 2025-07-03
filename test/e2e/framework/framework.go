package framework

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/mock/server"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/process"
)

type Options struct {
	TotalParallelNode int
	CurrentNodeIndex  int
	FromPortIndex     int
	ToPortIndex       int
}

type Framework struct {
	TempDirectory     string
	usedPorts         map[string]int
	allocatedPorts    []int
	portAllocator     *port.Allocator
	mockServers       *MockServers
	cleanupHandle     CleanupActionHandle
	beforeEachStarted bool

	serverConfPaths   []string
	serverProcesses   []*process.Process
	clientConfPaths   []string
	clientProcesses   []*process.Process

	servers         []server.Server
	configFileIndex int64
	osEnvs          []string
}

func NewDefaultFramework() *Framework {
	suiteCfg, _ := ginkgo.GinkgoConfiguration()
	return NewFramework(Options{
		TotalParallelNode: suiteCfg.ParallelTotal,
		CurrentNodeIndex:  suiteCfg.ParallelProcess,
		FromPortIndex:     10000,
		ToPortIndex:       30000,
	})
}

func NewFramework(opt Options) *Framework {
	f := &Framework{
		usedPorts:     make(map[string]int),
		portAllocator: port.NewAllocator(opt.FromPortIndex, opt.ToPortIndex, opt.TotalParallelNode, opt.CurrentNodeIndex-1),
	}

	ginkgo.BeforeEach(f.BeforeEach)
	ginkgo.AfterEach(f.AfterEach)
	return f
}

// BeforeEach sets up temp directory, mock servers, and allocates base ports.
func (f *Framework) BeforeEach() {
	f.beforeEachStarted = true
	f.cleanupHandle = AddCleanupAction(f.AfterEach)

	dir, err := os.MkdirTemp(os.TempDir(), "frp-e2e-test-*")
	ExpectNoError(err)
	f.TempDirectory = dir

	f.mockServers = NewMockServers(f.portAllocator)
	ExpectNoError(f.mockServers.Run())

	for name, val := range f.mockServers.GetTemplateParams() {
		if port, ok := val.(int); ok {
			f.usedPorts[name] = port
		}
	}
}

// AfterEach performs cleanup: stopping processes, removing temp files, releasing ports.
func (f *Framework) AfterEach() {
	if !f.beforeEachStarted {
		return
	}

	RemoveCleanupAction(f.cleanupHandle)
	f.stopProcesses()
	f.cleanupTempDirectory()
	f.releaseAllPorts()
	f.osEnvs = nil
}

func (f *Framework) stopProcesses() {
	for _, proc := range f.serverProcesses {
		_ = proc.Stop()
		printDebugOutput(proc)
	}
	for _, proc := range f.clientProcesses {
		_ = proc.Stop()
		printDebugOutput(proc)
	}
	f.serverProcesses = nil
	f.clientProcesses = nil

	f.mockServers.Close()
	for _, s := range f.servers {
		s.Close()
	}
}

func printDebugOutput(p *process.Process) {
	if TestContext.Debug || ginkgo.CurrentSpecReport().Failed() {
		fmt.Println(p.ErrorOutput())
		fmt.Println(p.StdOutput())
	}
}

func (f *Framework) cleanupTempDirectory() {
	if f.TempDirectory != "" {
		_ = os.RemoveAll(f.TempDirectory)
		f.TempDirectory = ""
	}
	f.serverConfPaths = nil
	f.clientConfPaths = nil
}

func (f *Framework) releaseAllPorts() {
	for _, port := range f.usedPorts {
		f.portAllocator.Release(port)
	}
	f.usedPorts = make(map[string]int)

	for _, port := range f.allocatedPorts {
		f.portAllocator.Release(port)
	}
	f.allocatedPorts = nil
}

// PortByName returns the allocated port for a given name.
func (f *Framework) PortByName(name string) int {
	return f.usedPorts[name]
}

// AllocPort reserves a new port and tracks it for cleanup.
func (f *Framework) AllocPort() int {
	port := f.portAllocator.Get()
	ExpectTrue(port > 0, "failed to allocate port")
	f.allocatedPorts = append(f.allocatedPorts, port)
	return port
}

func (f *Framework) ReleasePort(port int) {
	f.portAllocator.Release(port)
}

// RunServer registers and starts a mock server.
func (f *Framework) RunServer(portName string, s server.Server) {
	f.servers = append(f.servers, s)
	if port := s.BindPort(); port > 0 && portName != "" {
		f.usedPorts[portName] = port
	}
	ExpectNoError(s.Run(), "RunServer failed for portName %s", portName)
}

func (f *Framework) SetEnvs(envs []string) {
	f.osEnvs = envs
}

// WriteTempFile writes `content` into a temp file under the framework's temp dir.
func (f *Framework) WriteTempFile(name, content string) string {
	path := filepath.Join(f.TempDirectory, name)
	ExpectNoError(os.WriteFile(path, []byte(content), 0o600))
	return path
}

// RenderTemplates renders Go templates and allocates required ports.
func (f *Framework) RenderTemplates(templates []string) ([]string, map[string]int, error) {
	ports, err := f.extractAndAllocPorts(templates)
	if err != nil {
		return nil, nil, err
	}

	params := f.mockServers.GetTemplateParams()
	for k, v := range ports {
		params[k] = v
	}
	for k, v := range f.usedPorts {
		params[k] = v
	}

	var rendered []string
	for _, tmpl := range templates {
		t, err := template.New("frp-e2e").Parse(tmpl)
		if err != nil {
			return nil, nil, err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, params); err != nil {
			return nil, nil, err
		}
		rendered = append(rendered, buf.String())
	}
	return rendered, ports, nil
}

var portRegex = regexp.MustCompile(`{{\s*\.Port[^}]*}}`)

// extractAndAllocPorts parses templates for named ports and allocates them.
func (f *Framework) extractAndAllocPorts(templates []string) (map[string]int, error) {
	ports := make(map[string]int)
	for _, t := range templates {
		for _, match := range portRegex.FindAllString(t, -1) {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{ ."), "}}"))
			ports[name] = 0
		}
	}

	defer func() {
		if r := recover(); r != nil {
			for _, port := range ports {
				f.portAllocator.Release(port)
			}
		}
	}()

	for name := range ports {
		p := f.portAllocator.GetByName(name)
		if p <= 0 {
			return nil, fmt.Errorf("failed to allocate port for %q", name)
		}
		ports[name] = p
	}
	return ports, nil
}
