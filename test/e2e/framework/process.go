package framework

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	flog "github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/pkg/process"
)

func GenerateConfigFile(path string, content string) error {
	return ioutil.WriteFile(path, []byte(content), 0666)
}

// RunProcesses run multiple processes from templates.
// The first template should always be frps.
func (f *Framework) RunProcesses(serverTemplates []string, clientTemplates []string) {
	templates := make([]string, 0, len(serverTemplates)+len(clientTemplates))
	for _, t := range serverTemplates {
		templates = append(templates, t)
	}
	for _, t := range clientTemplates {
		templates = append(templates, t)
	}
	outs, ports, err := f.RenderTemplates(templates)
	ExpectNoError(err)
	ExpectTrue(len(templates) > 0)

	for name, port := range ports {
		f.usedPorts[name] = port
	}

	for i := range serverTemplates {
		path := filepath.Join(f.TempDirectory, fmt.Sprintf("frp-e2e-server-%d", i))
		err = ioutil.WriteFile(path, []byte(outs[i]), 0666)
		ExpectNoError(err)
		flog.Trace("[%s] %s", path, outs[i])
		p := process.New(TestContext.FRPServerPath, []string{"-c", path})
		f.serverConfPaths = append(f.serverConfPaths, path)
		f.serverProcesses = append(f.serverProcesses, p)
		err = p.Start()
		ExpectNoError(err)
	}
	time.Sleep(time.Second)

	for i := range clientTemplates {
		index := i + len(serverTemplates)
		path := filepath.Join(f.TempDirectory, fmt.Sprintf("frp-e2e-client-%d", i))
		err = ioutil.WriteFile(path, []byte(outs[index]), 0666)
		ExpectNoError(err)
		flog.Trace("[%s] %s", path, outs[index])
		p := process.New(TestContext.FRPClientPath, []string{"-c", path})
		f.clientConfPaths = append(f.clientConfPaths, path)
		f.clientProcesses = append(f.clientProcesses, p)
		err = p.Start()
		ExpectNoError(err)
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
}
