package apiauth

import (
	"net/url"
	"testing"
)

func TestSignature(t *testing.T) {
	appsecret := "beego secret"
	method := "GET"
	RequestURL := "http://localhost/test/url"
	params := make(url.Values)
	params.Add("arg1", "hello")
	params.Add("arg2", "beego")

	signature := "mFdpvLh48ca4mDVEItE9++AKKQ/IVca7O/ZyyB8hR58="
	if Signature(appsecret, method, params, RequestURL) != signature {
		t.Error("Signature error")
	}
}
