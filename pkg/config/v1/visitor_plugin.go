// Copyright 2025 The frp Authors
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
	"reflect"

	"github.com/fatedier/frp/pkg/util/jsonx"
)

const (
	VisitorPluginVirtualNet = "virtual_net"
)

var visitorPluginOptionsTypeMap = map[string]reflect.Type{
	VisitorPluginVirtualNet: reflect.TypeFor[VirtualNetVisitorPluginOptions](),
}

type VisitorPluginOptions interface {
	Complete()
	Clone() VisitorPluginOptions
}

type TypedVisitorPluginOptions struct {
	Type string `json:"type"`
	VisitorPluginOptions
}

func (c TypedVisitorPluginOptions) Clone() TypedVisitorPluginOptions {
	out := c
	if c.VisitorPluginOptions != nil {
		out.VisitorPluginOptions = c.VisitorPluginOptions.Clone()
	}
	return out
}

func (c *TypedVisitorPluginOptions) UnmarshalJSON(b []byte) error {
	decoded, err := DecodeVisitorPluginOptionsJSON(b, DecodeOptions{})
	if err != nil {
		return err
	}
	*c = decoded
	return nil
}

func (c *TypedVisitorPluginOptions) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(c.VisitorPluginOptions)
}

type VirtualNetVisitorPluginOptions struct {
	Type          string `json:"type"`
	DestinationIP string `json:"destinationIP"`
}

func (o *VirtualNetVisitorPluginOptions) Complete() {}

func (o *VirtualNetVisitorPluginOptions) Clone() VisitorPluginOptions {
	if o == nil {
		return nil
	}
	out := *o
	return &out
}
