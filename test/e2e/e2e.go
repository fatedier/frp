package e2e

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/framework"
)

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	setupSuite()
	return nil
}, func(data []byte) {
	// Run on all Ginkgo nodes
	setupSuitePerGinkgoNode()
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	CleanupSuite()
}, func() {
	AfterSuiteActions()
})

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(framework.Fail)

	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	suiteConfig.EmitSpecProgress = true
	// Randomize specs as well as suites
	suiteConfig.RandomizeAllSpecs = true

	log.Infof("Starting e2e run %q on Ginkgo node %d of total %d",
		framework.RunID, suiteConfig.ParallelProcess, suiteConfig.ParallelTotal)
	ginkgo.RunSpecs(t, "frp e2e suite", suiteConfig, reporterConfig)
}

// setupSuite is the boilerplate that can be used to setup ginkgo test suites, on the SynchronizedBeforeSuite step.
// There are certain operations we only want to run once per overall test invocation
// (such as deleting old namespaces, or verifying that all system pods are running.
// Because of the way Ginkgo runs tests in parallel, we must use SynchronizedBeforeSuite
// to ensure that these operations only run on the first parallel Ginkgo node.
//
// This function takes two parameters: one function which runs on only the first Ginkgo node,
// returning an opaque byte array, and then a second function which runs on all Ginkgo nodes,
// accepting the byte array.
func setupSuite() {
	// Run only on Ginkgo node 1
}

// setupSuitePerGinkgoNode is the boilerplate that can be used to setup ginkgo test suites, on the SynchronizedBeforeSuite step.
// There are certain operations we only want to run once per overall test invocation on each Ginkgo node
// such as making some global variables accessible to all parallel executions
// Because of the way Ginkgo runs tests in parallel, we must use SynchronizedBeforeSuite
// Ref: https://onsi.github.io/ginkgo/#parallel-specs
func setupSuitePerGinkgoNode() {
	// config.GinkgoConfig.ParallelNode
}
