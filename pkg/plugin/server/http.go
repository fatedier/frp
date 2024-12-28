// Copyright 2019 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugin

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type httpPlugin struct {
	options v1.HTTPPluginOptions

	url    string
	client *http.Client
}

func NewHTTPPluginOptions(options v1.HTTPPluginOptions) Plugin {
	url := fmt.Sprintf("%s%s", options.Addr, options.Path)

	var client *http.Client
	if strings.HasPrefix(url, "https://") {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !options.TLSVerify},
		}
		client = &http.Client{Transport: tr}
	} else {
		client = &http.Client{}
	}

	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	return &httpPlugin{
		options: options,
		url:     url,
		client:  client,
	}
}

func (p *httpPlugin) Name() string {
	return p.options.Name
}

func (p *httpPlugin) IsSupport(op string) bool {
	for _, v := range p.options.Ops {
		if v == op {
			return true
		}
	}
	return false
}

func (p *httpPlugin) Handle(ctx context.Context, op string, content any) (*Response, any, error) {
	r := &Request{
		Version: APIVersion,
		Op:      op,
		Content: content,
	}
	var res Response
	res.Content = reflect.New(reflect.TypeOf(content)).Interface()
	if err := p.do(ctx, r, &res); err != nil {
		return nil, nil, err
	}
	return &res, res.Content, nil
}

func (p *httpPlugin) do(ctx context.Context, r *Request, res *Response) error {
	buf, err := json.Marshal(r)
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("version", r.Version)
	v.Set("op", r.Op)
	req, err := http.NewRequest("POST", p.url+"?"+v.Encode(), bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("X-Frp-Reqid", GetReqidFromContext(ctx))
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("do http request error code: %d", resp.StatusCode)
	}
	buf, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, res)
}
