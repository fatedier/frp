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

package ssh

import (
	"github.com/fatedier/frp/client/proxy"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func createSuccessInfo(user string, pc v1.ProxyConfigurer, ps *proxy.WorkingStatus) string {
	base := pc.GetBaseConfig()
	out := "\n"
	out += "frp (via SSH) (Ctrl+C to quit)\n\n"
	out += "User: " + user + "\n"
	out += "ProxyName: " + base.Name + "\n"
	out += "Type: " + base.Type + "\n"
	out += "RemoteAddress: " + ps.RemoteAddr + "\n"
	return out
}
