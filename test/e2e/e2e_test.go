package e2e

import (
	"flag"
	"fmt"
	"os"
	"testing"

	_ "github.com/onsi/ginkgo"

	"github.com/fatedier/frp/pkg/util/log"
	// test source
	_ "github.com/fatedier/frp/test/e2e/basic"
	_ "github.com/fatedier/frp/test/e2e/features"
	"github.com/fatedier/frp/test/e2e/framework"
	_ "github.com/fatedier/frp/test/e2e/plugin"
)

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	framework.RegisterCommonFlags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	// Register test flags, then parse flags.
	handleFlags()

	if err := framework.ValidateTestContext(&framework.TestContext); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log.InitLog("console", "", framework.TestContext.LogLevel, 0, true)
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
