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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

const (
	VisitorPluginVirtualNet = "virtual_net"
)

var visitorPluginOptionsTypeMap = map[string]reflect.Type{
	VisitorPluginVirtualNet: reflect.TypeOf(VirtualNetVisitorPluginOptions{}),
}

type VisitorPluginOptions interface {
	Complete()
}

type TypedVisitorPluginOptions struct {
	Type string `json:"type"`
	VisitorPluginOptions
}

func (c *TypedVisitorPluginOptions) UnmarshalJSON(b []byte) error {
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
		return errors.New("visitor plugin type is empty")
	}

	v, ok := visitorPluginOptionsTypeMap[typeStruct.Type]
	if !ok {
		return fmt.Errorf("unknown visitor plugin type: %s", typeStruct.Type)
	}
	options := reflect.New(v).Interface().(VisitorPluginOptions)

	decoder := json.NewDecoder(bytes.NewBuffer(b))
	if DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(options); err != nil {
		return fmt.Errorf("unmarshal VisitorPluginOptions error: %v", err)
	}
	c.VisitorPluginOptions = options
	return nil
}

func (c *TypedVisitorPluginOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.VisitorPluginOptions)
}

type VirtualNetVisitorPluginOptions struct {
	Type          string `json:"type"`
	DestinationIP string `json:"destinationIP"`
}

func (o *VirtualNetVisitorPluginOptions) Complete() {}
