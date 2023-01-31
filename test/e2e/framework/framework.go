package framework

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"

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
	TempDirectory string

	// ports used in this framework indexed by port name.
	usedPorts map[string]int

	// record ports alloced by this framework and release them after each test
	allocedPorts []int

	// portAllocator to alloc port for this test case.
	portAllocator *port.Allocator

	// Multiple default mock servers used for e2e testing.
	mockServers *MockServers

	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle

	// beforeEachStarted indicates that BeforeEach has started
	beforeEachStarted bool

	serverConfPaths []string
	serverProcesses []*process.Process
	clientConfPaths []string
	clientProcesses []*process.Process

	// Manual registered mock servers.
	servers []server.Server

	// used to generate unique config file name.
	configFileIndex int64

	// envs used to start processes, the form is `key=value`.
	osEnvs []string
}

func NewDefaultFramework() *Framework {
	options := Options{
		TotalParallelNode: config.GinkgoConfig.ParallelTotal,
		CurrentNodeIndex:  config.GinkgoConfig.ParallelNode,
		FromPortIndex:     20000,
		ToPortIndex:       50000,
	}
	return NewFramework(options)
}

func NewFramework(opt Options) *Framework {
	f := &Framework{
		portAllocator: port.NewAllocator(opt.FromPortIndex, opt.ToPortIndex, opt.TotalParallelNode, opt.CurrentNodeIndex-1),
		usedPorts:     make(map[string]int),
	}

	ginkgo.BeforeEach(f.BeforeEach)
	ginkgo.AfterEach(f.AfterEach)
	return f
}

// BeforeEach create a temp directory.
func (f *Framework) BeforeEach() {
	f.beforeEachStarted = true

	f.cleanupHandle = AddCleanupAction(f.AfterEach)

	dir, err := os.MkdirTemp(os.TempDir(), "frp-e2e-test-*")
	ExpectNoError(err)
	f.TempDirectory = dir

	f.mockServers = NewMockServers(f.portAllocator)
	if err := f.mockServers.Run(); err != nil {
		Failf("%v", err)
	}

	params := f.mockServers.GetTemplateParams()
	for k, v := range params {
		switch t := v.(type) {
		case int:
			f.usedPorts[k] = t
		default:
		}
	}
}

func (f *Framework) AfterEach() {
	if !f.beforeEachStarted {
		return
	}

	RemoveCleanupAction(f.cleanupHandle)

	// stop processor
	for _, p := range f.serverProcesses {
		_ = p.Stop()
		if TestContext.Debug {
			fmt.Println(p.ErrorOutput())
			fmt.Println(p.StdOutput())
		}
	}
	for _, p := range f.clientProcesses {
		_ = p.Stop()
		if TestContext.Debug {
			fmt.Println(p.ErrorOutput())
			fmt.Println(p.StdOutput())
		}
	}
	f.serverProcesses = nil
	f.clientProcesses = nil

	// close default mock servers
	f.mockServers.Close()

	// close manual registered mock servers
	for _, s := range f.servers {
		s.Close()
	}

	// clean directory
	os.RemoveAll(f.TempDirectory)
	f.TempDirectory = ""
	f.serverConfPaths = []string{}
	f.clientConfPaths = []string{}

	// release used ports
	for _, port := range f.usedPorts {
		f.portAllocator.Release(port)
	}
	f.usedPorts = make(map[string]int)

	// release alloced ports
	for _, port := range f.allocedPorts {
		f.portAllocator.Release(port)
	}
	f.allocedPorts = make([]int, 0)

	// clear os envs
	f.osEnvs = make([]string, 0)
}

var portRegex = regexp.MustCompile(`{{ \.Port.*? }}`)

// RenderPortsTemplate render templates with ports.
//
// Local: {{ .Port1 }}
// Target: {{ .Port2 }}
//
// return rendered content and all allocated ports.
func (f *Framework) genPortsFromTemplates(templates []string) (ports map[string]int, err error) {
	ports = make(map[string]int)
	for _, t := range templates {
		arrs := portRegex.FindAllString(t, -1)
		for _, str := range arrs {
			str = strings.TrimPrefix(str, "{{ .")
			str = strings.TrimSuffix(str, " }}")
			str = strings.TrimSpace(str)
			ports[str] = 0
		}
	}
	defer func() {
		if err != nil {
			for _, port := range ports {
				f.portAllocator.Release(port)
			}
		}
	}()

	for name := range ports {
		port := f.portAllocator.GetByName(name)
		if port <= 0 {
			return nil, fmt.Errorf("can't allocate port")
		}
		ports[name] = port
	}
	return
}

// RenderTemplates alloc all ports for port names placeholder.
func (f *Framework) RenderTemplates(templates []string) (outs []string, ports map[string]int, err error) {
	ports, err = f.genPortsFromTemplates(templates)
	if err != nil {
		return
	}

	params := f.mockServers.GetTemplateParams()
	for name, port := range ports {
		params[name] = port
	}

	for name, port := range f.usedPorts {
		params[name] = port
	}

	for _, t := range templates {
		tmpl, err := template.New("").Parse(t)
		if err != nil {
			return nil, nil, err
		}
		buffer := bytes.NewBuffer(nil)
		if err = tmpl.Execute(buffer, params); err != nil {
			return nil, nil, err
		}
		outs = append(outs, buffer.String())
	}
	return
}

func (f *Framework) PortByName(name string) int {
	return f.usedPorts[name]
}

func (f *Framework) AllocPort() int {
	port := f.portAllocator.Get()
	ExpectTrue(port > 0, "alloc port failed")
	f.allocedPorts = append(f.allocedPorts, port)
	return port
}

func (f *Framework) ReleasePort(port int) {
	f.portAllocator.Release(port)
}

func (f *Framework) RunServer(portName string, s server.Server) {
	f.servers = append(f.servers, s)
	if s.BindPort() > 0 && portName != "" {
		f.usedPorts[portName] = s.BindPort()
	}
	err := s.Run()
	ExpectNoError(err, "RunServer: with PortName %s", portName)
}

func (f *Framework) SetEnvs(envs []string) {
	f.osEnvs = envs
}

func (f *Framework) WriteTempFile(name string, content string) string {
	filePath := filepath.Join(f.TempDirectory, name)
	err := os.WriteFile(filePath, []byte(content), 0o766)
	ExpectNoError(err)
	return filePath
}
