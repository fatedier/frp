package framework

import (
	"time"

	"github.com/fatedier/frp/test/e2e/pkg/request"
)

func ExpectTCPReuqest(port int, in, out []byte, timeout time.Duration) {
	res, err := request.SendTCPRequest(port, in, timeout)
	ExpectNoError(err)
	ExpectEqual(string(out), res)
}
