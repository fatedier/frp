package features

import (
	"github.com/fatedier/frp/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Real IP]", func() {
	f := framework.NewDefaultFramework()

	It("HTTP X-Forwarded-For", func() {
		// TODO
		_ = f
	})

	It("Proxy Protocol", func() {
		// TODO
	})
})
