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

package validation

import (
	"errors"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func ValidateClientPluginOptions(c v1.ClientPluginOptions) error {
	switch v := c.(type) {
	case *v1.HTTP2HTTPSPluginOptions:
		return validateHTTP2HTTPSPluginOptions(v)
	case *v1.HTTPS2HTTPPluginOptions:
		return validateHTTPS2HTTPPluginOptions(v)
	case *v1.HTTPS2HTTPSPluginOptions:
		return validateHTTPS2HTTPSPluginOptions(v)
	case *v1.StaticFilePluginOptions:
		return validateStaticFilePluginOptions(v)
	case *v1.UnixDomainSocketPluginOptions:
		return validateUnixDomainSocketPluginOptions(v)
	case *v1.TLS2RawPluginOptions:
		return validateTLS2RawPluginOptions(v)
	}
	return nil
}

func validateHTTP2HTTPSPluginOptions(c *v1.HTTP2HTTPSPluginOptions) error {
	if c.LocalAddr == "" {
		return errors.New("localAddr is required")
	}
	return nil
}

func validateHTTPS2HTTPPluginOptions(c *v1.HTTPS2HTTPPluginOptions) error {
	if c.LocalAddr == "" {
		return errors.New("localAddr is required")
	}
	return nil
}

func validateHTTPS2HTTPSPluginOptions(c *v1.HTTPS2HTTPSPluginOptions) error {
	if c.LocalAddr == "" {
		return errors.New("localAddr is required")
	}
	return nil
}

func validateStaticFilePluginOptions(c *v1.StaticFilePluginOptions) error {
	if c.LocalPath == "" {
		return errors.New("localPath is required")
	}
	return nil
}

func validateUnixDomainSocketPluginOptions(c *v1.UnixDomainSocketPluginOptions) error {
	if c.UnixPath == "" {
		return errors.New("unixPath is required")
	}
	return nil
}

func validateTLS2RawPluginOptions(c *v1.TLS2RawPluginOptions) error {
	if c.LocalAddr == "" {
		return errors.New("localAddr is required")
	}
	return nil
}
