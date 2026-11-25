package compatibility

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/process"
)

type compatTestContext struct {
	CurrentFRPSPath   string
	CurrentFRPCPath   string
	BaselineFRPSPath  string
	BaselineFRPCPath  string
	BaselineVersion   string
	LogLevel          string
	Debug             bool
	RunCurrentCurrent bool
}

var compatCtx compatTestContext

func registerFlags(flags *flag.FlagSet) {
	flags.StringVar(&compatCtx.CurrentFRPSPath, "current-frps-path", "../../bin/frps", "The current frps binary to use.")
	flags.StringVar(&compatCtx.CurrentFRPCPath, "current-frpc-path", "../../bin/frpc", "The current frpc binary to use.")
	flags.StringVar(&compatCtx.BaselineFRPSPath, "baseline-frps-path", "", "The baseline frps binary to use.")
	flags.StringVar(&compatCtx.BaselineFRPCPath, "baseline-frpc-path", "", "The baseline frpc binary to use.")
	flags.StringVar(&compatCtx.BaselineVersion, "baseline-version", "custom", "The baseline version label for reporting.")
	flags.StringVar(&compatCtx.LogLevel, "log-level", "debug", "Log level.")
	flags.BoolVar(&compatCtx.Debug, "debug", false, "Enable debug mode to print detailed info.")
	flags.BoolVar(&compatCtx.RunCurrentCurrent, "run-current-current", true, "Run current frps/current frpc sanity checks.")
}

func validateCompatContext(t *compatTestContext) error {
	paths := map[string]string{
		"current-frps-path":  t.CurrentFRPSPath,
		"current-frpc-path":  t.CurrentFRPCPath,
		"baseline-frps-path": t.BaselineFRPSPath,
		"baseline-frpc-path": t.BaselineFRPCPath,
	}
	for name, path := range paths {
		if path == "" {
			return fmt.Errorf("%s can't be empty", name)
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("load %s error: %v", name, err)
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	registerFlags(flag.CommandLine)
	flag.Parse()

	if err := validateCompatContext(&compatCtx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	framework.TestContext.Debug = compatCtx.Debug
	framework.TestContext.LogLevel = compatCtx.LogLevel
	log.InitLogger("console", compatCtx.LogLevel, 0, true)
	os.Exit(m.Run())
}

func TestCompatibilityE2E(t *testing.T) {
	gomega.RegisterFailHandler(framework.Fail)

	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	suiteConfig.EmitSpecProgress = true
	suiteConfig.RandomizeAllSpecs = true
	ginkgo.RunSpecs(t, "frp compatibility e2e suite", suiteConfig, reporterConfig)
}

var _ = ginkgo.Describe("[Compatibility: WireProtocol]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("current frps and current frpc support v1 and v2", func() {
		if !compatCtx.RunCurrentCurrent {
			ginkgo.Skip("current/current sanity checks already ran")
		}

		webPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
webServer.port = %d
		`, webPort)

		v1PortName := port.GenName("CompatCurrentV1")
		v1ClientConf := tcpClientConfig("compat-current-v1", v1PortName, `
clientID = "compat-current-v1"
transport.wireProtocol = "v1"
`)

		v2PortName := port.GenName("CompatCurrentV2")
		v2ClientConf := tcpClientConfig("compat-current-v2", v2PortName, `
clientID = "compat-current-v2"
transport.wireProtocol = "v2"
`)

		f.RunProcessesWithBinaries(
			compatCtx.CurrentFRPSPath,
			compatCtx.CurrentFRPCPath,
			serverConf,
			[]string{v1ClientConf, v2ClientConf},
		)

		framework.NewRequestExpect(f).PortName(v1PortName).Ensure()
		framework.NewRequestExpect(f).PortName(v2PortName).Ensure()
		expectClientWireProtocol(webPort, "compat-current-v1", "v1")
		expectClientWireProtocol(webPort, "compat-current-v2", "v2")
	})

	ginkgo.It("current frps accepts baseline frpc using v1", func() {
		webPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
webServer.port = %d
`, webPort)

		portName := port.GenName("CompatBaselineFRPC")
		clientConf := tcpClientConfig("tcp", portName, "")

		f.RunProcessesWithBinaries(
			compatCtx.CurrentFRPSPath,
			compatCtx.BaselineFRPCPath,
			serverConf,
			[]string{clientConf},
		)

		framework.NewRequestExpect(f).PortName(portName).Ensure()
		expectSingleClientWireProtocol(webPort, "v1")
	})

	ginkgo.It("baseline frps accepts current frpc defaulting to v1", func() {
		portName := port.GenName("CompatBaselineFRPS")
		clientConf := tcpClientConfig("tcp", portName, "")

		f.RunProcessesWithBinaries(
			compatCtx.BaselineFRPSPath,
			compatCtx.CurrentFRPCPath,
			consts.DefaultServerConfig,
			[]string{clientConf},
		)

		framework.NewRequestExpect(f).PortName(portName).Ensure()
	})

	ginkgo.It("baseline frps handles current frpc forced to v2 according to baseline support", func() {
		portName := port.GenName("CompatBaselineFRPSForcedV2")
		clientConf := tcpClientConfig("tcp", portName, `
transport.wireProtocol = "v2"
`)

		_, clientProcesses := f.RunProcessesWithBinaries(
			compatCtx.BaselineFRPSPath,
			compatCtx.CurrentFRPCPath,
			consts.DefaultServerConfig,
			[]string{clientConf},
		)

		// frp v0.69.0 added control connection wireProtocol v2 support, so
		// baseline frps v0.69.0 and newer should accept a current frpc forced
		// to v2. Older known baselines still must reject the unsupported protocol.
		// For custom baselines, the version is unknown and the binary may be either
		// side of the support boundary, so this versioned expectation is skipped.
		supportsV2, knownVersion := baselineSupportsControlWireProtocolV2(compatCtx.BaselineVersion)
		if !knownVersion {
			ginkgo.Skip(fmt.Sprintf("baseline version %q is not semver; skip versioned forced-v2 expectation", compatCtx.BaselineVersion))
		}
		if supportsV2 {
			framework.NewRequestExpect(f).PortName(portName).Ensure()
			return
		}

		expectProcessExit(clientProcesses[0], 5*time.Second)
		framework.NewRequestExpect(f).PortName(portName).ExpectError(true).Ensure()
	})
})

var _ = ginkgo.Describe("[Compatibility: UDPDefault]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("current frps exchanges UDP with baseline frpc using the default wire protocol", func() {
		portName := port.GenName("CompatDefaultUDPBaselineFRPC")
		f.RunProcessesWithBinaries(
			compatCtx.CurrentFRPSPath,
			compatCtx.BaselineFRPCPath,
			consts.DefaultServerConfig,
			[]string{udpClientConfig(portName, "")},
		)
		framework.NewRequestExpect(f).Protocol("udp").PortName(portName).Ensure()
	})

	ginkgo.It("baseline frps exchanges UDP with current frpc defaulting to v1", func() {
		portName := port.GenName("CompatDefaultUDPBaselineFRPS")
		f.RunProcessesWithBinaries(
			compatCtx.BaselineFRPSPath,
			compatCtx.CurrentFRPCPath,
			consts.DefaultServerConfig,
			[]string{udpClientConfig(portName, "")},
		)
		framework.NewRequestExpect(f).Protocol("udp").PortName(portName).Ensure()
	})
})

var _ = ginkgo.Describe("[Compatibility: UDPWireV2]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.BeforeEach(func() {
		supportsV2, knownVersion := baselineSupportsUDPWireProtocolV2(compatCtx.BaselineVersion)
		if !knownVersion {
			ginkgo.Skip(fmt.Sprintf("baseline version %q has unknown ordinary UDP wire v2 framing support", compatCtx.BaselineVersion))
		}
		if !supportsV2 {
			// v0.69.0 supports control v2 but still uses legacy ordinary UDP framing.
			// Mixing that early v2 implementation with the v0.69.1 framing was not supported.
			ginkgo.Skip(fmt.Sprintf("baseline %q predates ordinary UDP wire v2 framing (v0.69.1); default UDP is tested separately",
				compatCtx.BaselineVersion))
		}
	})

	// The negotiated payload codec may be JSON or binary-v1; both use v2 framing.
	ginkgo.It("current frps exchanges v2-framed UDP with baseline frpc", func() {
		portName := port.GenName("CompatV2UDPBaselineFRPC")
		clientConf := udpClientConfig(portName, `transport.wireProtocol = "v2"`)
		f.RunProcessesWithBinaries(
			compatCtx.CurrentFRPSPath,
			compatCtx.BaselineFRPCPath,
			consts.DefaultServerConfig,
			[]string{clientConf},
		)
		framework.NewRequestExpect(f).Protocol("udp").PortName(portName).Ensure()
	})

	ginkgo.It("current frpc exchanges v2-framed UDP with baseline frps", func() {
		portName := port.GenName("CompatV2UDPBaselineFRPS")
		clientConf := udpClientConfig(portName, `transport.wireProtocol = "v2"`)
		f.RunProcessesWithBinaries(
			compatCtx.BaselineFRPSPath,
			compatCtx.CurrentFRPCPath,
			consts.DefaultServerConfig,
			[]string{clientConf},
		)
		framework.NewRequestExpect(f).Protocol("udp").PortName(portName).Ensure()
	})
})

func tcpClientConfig(proxyName string, remotePortName string, extra string) string {
	return fmt.Sprintf(`
serverAddr = "127.0.0.1"
serverPort = {{ .%s }}
loginFailExit = true
log.level = "trace"
%s

[[proxies]]
name = "%s"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, consts.PortServerName, extra, proxyName, framework.TCPEchoServerPort, remotePortName)
}

func udpClientConfig(remotePortName string, extra string) string {
	return fmt.Sprintf(`
serverAddr = "127.0.0.1"
serverPort = {{ .%s }}
loginFailExit = true
log.level = "trace"
%s

[[proxies]]
name = "udp"
type = "udp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, consts.PortServerName, extra, framework.UDPEchoServerPort, remotePortName)
}

func expectProcessExit(p *process.Process, timeout time.Duration) {
	select {
	case <-p.Done():
	case <-time.After(timeout):
		framework.Failf("process did not exit within %s; output:\n%s", timeout, p.Output())
	}
}

func baselineSupportsControlWireProtocolV2(version string) (supports bool, known bool) {
	return baselineVersionAtLeast(version, [3]int{0, 69, 0})
}

func baselineSupportsUDPWireProtocolV2(version string) (supports bool, known bool) {
	return baselineVersionAtLeast(version, [3]int{0, 69, 1})
}

func baselineVersionAtLeast(version string, minimum [3]int) (supports bool, known bool) {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, false
	}

	return compareSemanticVersion(major, minor, patch, minimum[0], minimum[1], minimum[2]) >= 0, true
}

func TestBaselineWireProtocolSupport(t *testing.T) {
	for _, tc := range []struct {
		version   string
		known     bool
		controlV2 bool
		udpV2     bool
	}{
		{version: "0.61.0", known: true},
		{version: "0.68.1", known: true},
		{version: "0.69.0", known: true, controlV2: true},
		{version: "v0.69.0", known: true, controlV2: true},
		{version: "0.69.1", known: true, controlV2: true, udpV2: true},
		{version: "v0.69.1", known: true, controlV2: true, udpV2: true},
		{version: "0.70.0", known: true, controlV2: true, udpV2: true},
		{version: "0.71.0", known: true, controlV2: true, udpV2: true},
		{version: "1.0.0", known: true, controlV2: true, udpV2: true},
		{version: "custom"},
		{version: "0.69"},
		{version: "0.69.x"},
		{version: "0.69.1-rc.1"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			if supports, known := baselineSupportsControlWireProtocolV2(tc.version); supports != tc.controlV2 || known != tc.known {
				t.Errorf("control v2 support = (%v, %v), want (%v, %v)", supports, known, tc.controlV2, tc.known)
			}
			if supports, known := baselineSupportsUDPWireProtocolV2(tc.version); supports != tc.udpV2 || known != tc.known {
				t.Errorf("UDP v2 framing support = (%v, %v), want (%v, %v)", supports, known, tc.udpV2, tc.known)
			}
		})
	}
}

func compareSemanticVersion(major, minor, patch int, baseMajor, baseMinor, basePatch int) int {
	if major != baseMajor {
		return major - baseMajor
	}
	if minor != baseMinor {
		return minor - baseMinor
	}
	return patch - basePatch
}

type wireClientInfo struct {
	ClientID     string `json:"clientID"`
	WireProtocol string `json:"wireProtocol"`
}

func expectSingleClientWireProtocol(webPort int, wireProtocol string) {
	clients := getWireClientInfos(webPort)
	framework.ExpectEqual(len(clients), 1)
	framework.ExpectEqual(clients[0].WireProtocol, wireProtocol)
}

func expectClientWireProtocol(webPort int, clientID string, wireProtocol string) {
	for _, client := range getWireClientInfos(webPort) {
		if client.ClientID == clientID {
			framework.ExpectEqual(client.WireProtocol, wireProtocol)
			return
		}
	}
	framework.Failf("client %q not found in /api/clients response", clientID)
}

func getWireClientInfos(webPort int) []wireClientInfo {
	client := http.Client{Timeout: consts.DefaultTimeout}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/clients", webPort))
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)

	content, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	var clients []wireClientInfo
	framework.ExpectNoError(json.Unmarshal(content, &clients))
	return clients
}
