package alils

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
)

// request sends a request to SLS.
func request(project *LogProject, method, uri string, headers map[string]string,
	body []byte) (resp *http.Response, err error) {

	// The caller should provide 'x-sls-bodyrawsize' header
	if _, ok := headers["x-sls-bodyrawsize"]; !ok {
		err = fmt.Errorf("Can't find 'x-sls-bodyrawsize' header")
		return
	}

	// SLS public request headers
	headers["Host"] = project.Name + "." + project.Endpoint
	headers["Date"] = nowRFC1123()
	headers["x-sls-apiversion"] = version
	headers["x-sls-signaturemethod"] = signatureMethod
	if body != nil {
		bodyMD5 := fmt.Sprintf("%X", md5.Sum(body))
		headers["Content-MD5"] = bodyMD5

		if _, ok := headers["Content-Type"]; !ok {
			err = fmt.Errorf("Can't find 'Content-Type' header")
			return
		}
	}

	// Calc Authorization
	// Authorization = "SLS <AccessKeyId>:<Signature>"
	digest, err := signature(project, method, uri, headers)
	if err != nil {
		return
	}
	auth := fmt.Sprintf("SLS %v:%v", project.AccessKeyId, digest)
	headers["Authorization"] = auth

	// Initialize http request
	reader := bytes.NewReader(body)
	urlStr := fmt.Sprintf("http://%v.%v%v", project.Name, project.Endpoint, uri)
	req, err := http.NewRequest(method, urlStr, reader)
	if err != nil {
		return
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	// Get ready to do request
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	return
}
