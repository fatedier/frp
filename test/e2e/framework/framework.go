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
	TempDirectory string

	// פורטים בשימוש במסגרת הזו לפי שם.
	usedPorts map[string]int

	// רישום הפורטים שהוקצו וישוחררו אחרי כל בדיקה.
	allocatedPorts []int

	// הקצאת פורטים למקרה בדיקה זה.
	portAllocator *port.Allocator

	// מספר שרתי הדמיה (mock) עבור בדיקות קצה.
	mockServers *MockServers

	// כדי לוודא שהמסגרת הזו מנקה אחריה, לא משנה מה קורה,
	// אנחנו מתקינים פעולה לניקוי לפני כל בדיקה ומסירים אותה אחרי.
	// במקרה של כישלון, הפונקציה AfterSuite תפעיל את כל פעולות הניקוי.
	cleanupHandle CleanupActionHandle

	// מציין ש-BeforeEach התחיל
	beforeEachStarted bool

	serverConfPaths []string
	serverProcesses []*process.Process
	clientConfPaths []string
	clientProcesses []*process.Process

	// שרתי mock שנרשמו באופן ידני.
	servers []server.Server

	// משמש ליצירת שם ייחודי לקובץ קונפיגורציה.
	configFileIndex int64

	// משתני סביבה לצורך הפעלת תהליכים, בפורמט `key=value`.
	osEnvs []string
}

func NewDefaultFramework() *Framework {
	suiteConfig, _ := ginkgo.GinkgoConfiguration()
	options := Options{
		TotalParallelNode: suiteConfig.ParallelTotal,
		CurrentNodeIndex:  suiteConfig.ParallelProcess,
		FromPortIndex:     10000,
		ToPortIndex:       30000,
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

// BeforeEach יוצר תיקייה זמנית.
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

	// עצירת התהליכים
	for _, p := range f.serverProcesses {
		_ = p.Stop()
		if TestContext.Debug || ginkgo.CurrentSpecReport().Failed() {
			fmt.Println(p.ErrorOutput())
			fmt.Println(p.StdOutput())
		}
	}
	for _, p := range f.clientProcesses {
		_ = p.Stop()
		if TestContext.Debug || ginkgo.CurrentSpecReport().Failed() {
			fmt.Println(p.ErrorOutput())
			fmt.Println(p.StdOutput())
		}
	}
	f.serverProcesses = nil
	f.clientProcesses = nil

	// סגירת שרתי mock ברירת מחדל
	f.mockServers.Close()

	// סגירת שרתי mock שנרשמו ידנית
	for _, s := range f.servers {
		s.Close()
	}

	// ניקוי התיקייה הזמנית
	os.RemoveAll(f.TempDirectory)
	f.TempDirectory = ""
	f.serverConfPaths = []string{}
	f.clientConfPaths = []string{}

	// שחרור פורטים בשימוש
	for _, port := range f.usedPorts {
		f.portAllocator.Release(port)
	}
	f.usedPorts = make(map[string]int)

	// שחרור פורטים שהוקצו
	for _, port := range f.allocatedPorts {
		f.portAllocator.Release(port)
	}
	f.allocatedPorts = make([]int, 0)

	// ניקוי משתני סביבה
	f.osEnvs = make([]string, 0)
}

var portRegex = regexp.MustCompile(`{{ \.Port.*? }}`)

// RenderPortsTemplate מרנדר תבניות עם פורטים.
//
// מקומי: {{ .Port1 }}
// יעד:   {{ .Port2 }}
//
// מחזיר תוכן מרונדר וכל הפורטים שהוקצו.
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
			return nil, fmt.Errorf("לא ניתן להקצות פורט")
		}
		ports[name] = port
	}
	return
}

// RenderTemplates מקצה פורטים לכל תבנית עם שמות פורטים.
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
		tmpl, err := template.New("frp-e2e").Parse(t)
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
	ExpectTrue(port > 0, "הקצאת פורט נכשלה")
	f.allocatedPorts = append(f.allocatedPorts, port)
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
	ExpectNoError(err, "RunServer: עם שם פורט %s", portName)
}

func (f *Framework) SetEnvs(envs []string) {
	f.osEnvs = envs
}

func (f *Framework) WriteTempFile(name string, content string) string {
	filePath := filepath.Join(f.TempDirectory, name)
	err := os.WriteFile(filePath, []byte(content), 0o600)
	ExpectNoError(err)
	return filePath
}
