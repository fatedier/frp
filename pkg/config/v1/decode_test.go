// Copyright 2026 The frp Authors
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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeProxyConfigurerJSON_StrictPluginUnknownFields(t *testing.T) {
	require := require.New(t)

	data := []byte(`{
		"name":"p1",
		"type":"tcp",
		"localPort":10080,
		"plugin":{
			"type":"http2https",
			"localAddr":"127.0.0.1:8080",
			"unknownInPlugin":"value"
		}
	}`)

	_, err := DecodeProxyConfigurerJSON(data, DecodeOptions{DisallowUnknownFields: false})
	require.NoError(err)

	_, err = DecodeProxyConfigurerJSON(data, DecodeOptions{DisallowUnknownFields: true})
	require.ErrorContains(err, "unknownInPlugin")
}

func TestDecodeVisitorConfigurerJSON_StrictPluginUnknownFields(t *testing.T) {
	require := require.New(t)

	data := []byte(`{
		"name":"v1",
		"type":"stcp",
		"serverName":"server",
		"bindPort":10081,
		"plugin":{
			"type":"virtual_net",
			"destinationIP":"10.0.0.1",
			"unknownInPlugin":"value"
		}
	}`)

	_, err := DecodeVisitorConfigurerJSON(data, DecodeOptions{DisallowUnknownFields: false})
	require.NoError(err)

	_, err = DecodeVisitorConfigurerJSON(data, DecodeOptions{DisallowUnknownFields: true})
	require.ErrorContains(err, "unknownInPlugin")
}

func TestDecodeClientConfigJSON_StrictUnknownProxyField(t *testing.T) {
	require := require.New(t)

	data := []byte(`{
		"serverPort":7000,
		"proxies":[
			{
				"name":"p1",
				"type":"tcp",
				"localPort":10080,
				"unknownField":"value"
			}
		]
	}`)

	_, err := DecodeClientConfigJSON(data, DecodeOptions{DisallowUnknownFields: false})
	require.NoError(err)

	_, err = DecodeClientConfigJSON(data, DecodeOptions{DisallowUnknownFields: true})
	require.ErrorContains(err, "unknownField")
}
