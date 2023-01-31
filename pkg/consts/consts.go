// Copyright 2016 fatedier, fatedier@gmail.com
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

package consts

var (
	// proxy status
	Idle    = "idle"
	Working = "working"
	Closed  = "closed"
	Online  = "online"
	Offline = "offline"

	// proxy type
	TCPProxy    = "tcp"
	UDPProxy    = "udp"
	TCPMuxProxy = "tcpmux"
	HTTPProxy   = "http"
	HTTPSProxy  = "https"
	STCPProxy   = "stcp"
	XTCPProxy   = "xtcp"
	SUDPProxy   = "sudp"

	// authentication method
	TokenAuthMethod = "token"
	OidcAuthMethod  = "oidc"

	// TCP multiplexer
	HTTPConnectTCPMultiplexer = "httpconnect"
)
