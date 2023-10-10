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

package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type Wrap struct {
	B   BandwidthQuantity `json:"b"`
	Int int               `json:"int"`
}

func TestBandwidthQuantity(t *testing.T) {
	require := require.New(t)

	var w Wrap
	err := json.Unmarshal([]byte(`{"b":"1KB","int":5}`), &w)
	require.NoError(err)
	require.EqualValues(1*KB, w.B.Bytes())

	buf, err := json.Marshal(&w)
	require.NoError(err)
	require.Equal(`{"b":"1KB","int":5}`, string(buf))
}

func TestPortsRangeSlice2String(t *testing.T) {
	require := require.New(t)

	ports := []PortsRange{
		{
			Start: 1000,
			End:   2000,
		},
		{
			Single: 3000,
		},
	}
	str := PortsRangeSlice(ports).String()
	require.Equal("1000-2000,3000", str)
}

func TestNewPortsRangeSliceFromString(t *testing.T) {
	require := require.New(t)

	ports, err := NewPortsRangeSliceFromString("1000-2000,3000")
	require.NoError(err)
	require.Equal([]PortsRange{
		{
			Start: 1000,
			End:   2000,
		},
		{
			Single: 3000,
		},
	}, ports)
}
