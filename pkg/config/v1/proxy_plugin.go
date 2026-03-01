// Copyright 2023 The frp Authors
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

package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/util/util"
)

const (
	PluginHTTP2HTTPS       = "http2https"
	PluginHTTPProxy        = "http_proxy"
	PluginHTTPS2HTTP       = "https2http"
	PluginHTTPS2HTTPS      = "https2https"
	PluginHTTP2HTTP        = "http2http"
	PluginSocks5           = "socks5"
	PluginStaticFile       = "static_file"
	PluginUnixDomainSocket = "unix_domain_socket"
	PluginTLS2Raw          = "tls2raw"
	PluginVirtualNet       = "virtual_net"
)

var clientPluginOptionsTypeMap = map[string]reflect.Type{
	PluginHTTP2HTTPS:       reflect.TypeOf(HTTP2HTTPSPluginOptions{}),
	PluginHTTPProxy:        reflect.TypeOf(HTTPProxyPluginOptions{}),
	PluginHTTPS2HTTP:       reflect.TypeOf(HTTPS2HTTPPluginOptions{}),
	PluginHTTPS2HTTPS:      reflect.TypeOf(HTTPS2HTTPSPluginOptions{}),
	PluginHTTP2HTTP:        reflect.TypeOf(HTTP2HTTPPluginOptions{}),
	PluginSocks5:           reflect.TypeOf(Socks5PluginOptions{}),
	PluginStaticFile:       reflect.TypeOf(StaticFilePluginOptions{}),
	PluginUnixDomainSocket: reflect.TypeOf(UnixDomainSocketPluginOptions{}),
	PluginTLS2Raw:          reflect.TypeOf(TLS2RawPluginOptions{}),
	PluginVirtualNet:       reflect.TypeOf(VirtualNetPluginOptions{}),
}

type ClientPluginOptions interface {
	Complete()
	Clone() ClientPluginOptions
}

type TypedClientPluginOptions struct {
	Type string `json:"type"`
	ClientPluginOptions
}

func (c TypedClientPluginOptions) Clone() TypedClientPluginOptions {
	out := c
	if c.ClientPluginOptions != nil {
		out.ClientPluginOptions = c.ClientPluginOptions.Clone()
	}
	return out
}

func (c *TypedClientPluginOptions) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		return nil
	}

	typeStruct := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &typeStruct); err != nil {
		return err
	}

	c.Type = typeStruct.Type
	if c.Type == "" {
		return errors.New("plugin type is empty")
	}

	v, ok := clientPluginOptionsTypeMap[typeStruct.Type]
	if !ok {
		return fmt.Errorf("unknown plugin type: %s", typeStruct.Type)
	}
	options := reflect.New(v).Interface().(ClientPluginOptions)

	decoder := json.NewDecoder(bytes.NewBuffer(b))
	if DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(options); err != nil {
		return fmt.Errorf("unmarshal ClientPluginOptions error: %v", err)
	}
	c.ClientPluginOptions = options
	return nil
}

func (c *TypedClientPluginOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ClientPluginOptions)
}

type HTTP2HTTPSPluginOptions struct {
	Type              string           `json:"type,omitempty"`
	LocalAddr         string           `json:"localAddr,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
}

func (o *HTTP2HTTPSPluginOptions) Complete() {}

func (o *HTTP2HTTPSPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	out.RequestHeaders = o.RequestHeaders.Clone()
	return &out
}

type HTTPProxyPluginOptions struct {
	Type         string `json:"type,omitempty"`
	HTTPUser     string `json:"httpUser,omitempty"`
	HTTPPassword string `json:"httpPassword,omitempty"`
}

func (o *HTTPProxyPluginOptions) Complete() {}

func (o *HTTPProxyPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	return &out
}

type HTTPS2HTTPPluginOptions struct {
	Type              string           `json:"type,omitempty"`
	LocalAddr         string           `json:"localAddr,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
	EnableHTTP2       *bool            `json:"enableHTTP2,omitempty"`
	CrtPath           string           `json:"crtPath,omitempty"`
	KeyPath           string           `json:"keyPath,omitempty"`
}

func (o *HTTPS2HTTPPluginOptions) Complete() {
	o.EnableHTTP2 = util.EmptyOr(o.EnableHTTP2, lo.ToPtr(true))
}

func (o *HTTPS2HTTPPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	out.RequestHeaders = o.RequestHeaders.Clone()
	out.EnableHTTP2 = util.ClonePtr(o.EnableHTTP2)
	return &out
}

type HTTPS2HTTPSPluginOptions struct {
	Type              string           `json:"type,omitempty"`
	LocalAddr         string           `json:"localAddr,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
	EnableHTTP2       *bool            `json:"enableHTTP2,omitempty"`
	CrtPath           string           `json:"crtPath,omitempty"`
	KeyPath           string           `json:"keyPath,omitempty"`
}

func (o *HTTPS2HTTPSPluginOptions) Complete() {
	o.EnableHTTP2 = util.EmptyOr(o.EnableHTTP2, lo.ToPtr(true))
}

func (o *HTTPS2HTTPSPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	out.RequestHeaders = o.RequestHeaders.Clone()
	out.EnableHTTP2 = util.ClonePtr(o.EnableHTTP2)
	return &out
}

type HTTP2HTTPPluginOptions struct {
	Type              string           `json:"type,omitempty"`
	LocalAddr         string           `json:"localAddr,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
}

func (o *HTTP2HTTPPluginOptions) Complete() {}

func (o *HTTP2HTTPPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	out.RequestHeaders = o.RequestHeaders.Clone()
	return &out
}

type Socks5PluginOptions struct {
	Type     string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (o *Socks5PluginOptions) Complete() {}

func (o *Socks5PluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	return &out
}

type StaticFilePluginOptions struct {
	Type         string `json:"type,omitempty"`
	LocalPath    string `json:"localPath,omitempty"`
	StripPrefix  string `json:"stripPrefix,omitempty"`
	HTTPUser     string `json:"httpUser,omitempty"`
	HTTPPassword string `json:"httpPassword,omitempty"`
}

func (o *StaticFilePluginOptions) Complete() {}

func (o *StaticFilePluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	return &out
}

type UnixDomainSocketPluginOptions struct {
	Type     string `json:"type,omitempty"`
	UnixPath string `json:"unixPath,omitempty"`
}

func (o *UnixDomainSocketPluginOptions) Complete() {}

func (o *UnixDomainSocketPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	return &out
}

type TLS2RawPluginOptions struct {
	Type      string `json:"type,omitempty"`
	LocalAddr string `json:"localAddr,omitempty"`
	CrtPath   string `json:"crtPath,omitempty"`
	KeyPath   string `json:"keyPath,omitempty"`
}

func (o *TLS2RawPluginOptions) Complete() {}

func (o *TLS2RawPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	return &out
}

type VirtualNetPluginOptions struct {
	Type string `json:"type,omitempty"`
}

func (o *VirtualNetPluginOptions) Complete() {}

func (o *VirtualNetPluginOptions) Clone() ClientPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	return &out
}
