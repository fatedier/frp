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

package legacy

import (
	"reflect"
	"testing"
)

// A comma-separated nat_hole_stun_server value may carry spaces around the
// entries — a natural ini style ("a:3478, b:3478") that strings.Split alone
// turns into entries with a leading space, which then fail address resolution.
func TestConvertClientCommonConfTrimsSTUNServerList(t *testing.T) {
	conf := ClientCommonConf{
		ServerAddr:        "example.com",
		NatHoleSTUNServer: "stun1.example.com:3478, stun2.example.com:3478",
	}
	out := Convert_ClientCommonConf_To_v1(&conf)

	want := []string{"stun1.example.com:3478", "stun2.example.com:3478"}
	if !reflect.DeepEqual([]string(out.NatHoleSTUNServer), want) {
		t.Fatalf("NatHoleSTUNServer = %#v, want %#v", out.NatHoleSTUNServer, want)
	}

	single := ClientCommonConf{
		ServerAddr:        "example.com",
		NatHoleSTUNServer: "stun.example.com:3478",
	}
	outSingle := Convert_ClientCommonConf_To_v1(&single)
	if !reflect.DeepEqual([]string(outSingle.NatHoleSTUNServer), []string{"stun.example.com:3478"}) {
		t.Fatalf("single-entry NatHoleSTUNServer = %#v, want the entry unchanged", outSingle.NatHoleSTUNServer)
	}
}
