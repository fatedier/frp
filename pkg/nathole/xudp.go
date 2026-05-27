// Copyright 2024 The frp Authors
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

package nathole

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// MultiSTUNResult holds the results from querying multiple STUN servers concurrently.
type MultiSTUNResult struct {
	Addrs     []string
	LocalAddr net.Addr
	// PortDelta is the observed port increment between successive STUN responses.
	// A value of 0 means no predictable pattern was detected.
	PortDelta int
}

// DiscoverMultiSTUN queries multiple STUN servers serially from the same local port
// to observe mapping rules and predict the next allocated port for Symmetric NATs.
// Requests are issued serially to avoid response demux issues on the shared UDP socket.
func DiscoverMultiSTUN(ctx context.Context, stunServers []string, localAddr string) (*MultiSTUNResult, error) {
	if len(stunServers) == 0 {
		return nil, fmt.Errorf("no STUN servers provided")
	}

	discoverConn, err := listen(localAddr)
	if err != nil {
		return nil, fmt.Errorf("listen error: %v", err)
	}
	defer discoverConn.Close()

	go discoverConn.readLoop()

	// Issue STUN requests serially to avoid response demux issues on the shared
	// messageChan. Each request/response pair completes before the next begins.
	allAddrs := make([]string, 0, len(stunServers)*2)
	for _, server := range stunServers {
		if ctx.Err() != nil {
			break
		}
		addrs, err := discoverConn.discoverFromStunServer(server)
		if err != nil {
			continue
		}
		allAddrs = append(allAddrs, addrs...)
	}

	if len(allAddrs) == 0 {
		return nil, fmt.Errorf("no addresses discovered from any STUN server")
	}

	portDelta := computePortDelta(allAddrs)

	return &MultiSTUNResult{
		Addrs:     allAddrs,
		LocalAddr: discoverConn.localAddr,
		PortDelta: portDelta,
	}, nil
}

// computePortDelta analyzes the port sequence from multiple STUN responses
// to detect a predictable increment pattern. Ports are sorted before analysis
// to handle non-deterministic response ordering.
func computePortDelta(addrs []string) int {
	ports := parsePorts(addrs)
	if len(ports) < 2 {
		return 0
	}

	// Sort ports to handle non-deterministic response ordering
	sort.Ints(ports)

	// Check if deltas are consistent
	delta := ports[1] - ports[0]
	if delta <= 0 {
		return 0
	}
	for i := 2; i < len(ports); i++ {
		if ports[i]-ports[i-1] != delta {
			return 0
		}
	}
	// Only report delta if it's within a reasonable range (1-100)
	if delta >= 1 && delta <= 100 {
		return delta
	}
	return 0
}

// parsePorts extracts port numbers from host:port address strings.
func parsePorts(addrs []string) []int {
	ports := make([]int, 0, len(addrs))
	for _, addr := range addrs {
		_, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			continue
		}
		port := 0
		fmt.Sscanf(portStr, "%d", &port)
		if port > 0 {
			ports = append(ports, port)
		}
	}
	return ports
}

// XUDPRendezvousExchange performs the realm rendezvous exchange for xudp mode.
// It sends the visitor/client message to the server and waits for the response.
// The server acts as a lightweight signaling relay using the shared token,
// then cleanly exits the data flow.
func XUDPRendezvousExchange(
	ctx context.Context,
	transporter transport.MessageTransporter,
	laneKey string,
	m msg.Message,
	timeout time.Duration,
) (*msg.NatHoleResp, error) {
	xl := xlog.FromContextSafe(ctx)
	xl.Tracef("xudp rendezvous exchange start, laneKey: %s", laneKey)

	resp, err := ExchangeInfo(ctx, transporter, laneKey, m, timeout)
	if err != nil {
		return nil, fmt.Errorf("xudp rendezvous exchange error: %v", err)
	}

	xl.Infof("xudp rendezvous exchange success, sid [%s], candidate addrs: %v", resp.Sid, resp.CandidateAddrs)
	return resp, nil
}

// XUDPMakeHole performs NAT hole punching for xudp mode with multi-STUN prediction.
// It extends the standard MakeHole with additional port prediction logic for Symmetric NATs.
//
// Note: portDelta is the locally observed allocation increment. It is applied to the
// peer's candidate addresses as a best-effort prediction. This works well when both
// sides exhibit similar NAT behavior (common with same-ISP or same-model routers).
// When deltas differ across peers, the prediction simply adds extra candidates without
// harm — MakeHole still succeeds via the standard addresses. Exchanging deltas through
// the rendezvous would improve accuracy but requires a wire protocol change, which is
// intentionally out of scope for backward compatibility.
func XUDPMakeHole(
	ctx context.Context,
	listenConn *net.UDPConn,
	resp *msg.NatHoleResp,
	key []byte,
	portDelta int,
) (*net.UDPConn, *net.UDPAddr, error) {
	xl := xlog.FromContextSafe(ctx)

	// If we have a predictable port delta, augment candidate addresses with predicted ports
	if portDelta > 0 && len(resp.CandidateAddrs) > 0 {
		predicted := predictNextPorts(resp.CandidateAddrs, portDelta, 3)
		xl.Debugf("xudp predicted additional candidate ports: %v", predicted)
		resp.CandidateAddrs = append(resp.CandidateAddrs, predicted...)
		// Deduplicate to avoid redundant packets
		resp.CandidateAddrs = dedup(resp.CandidateAddrs)
	}

	// Use the standard MakeHole mechanism with augmented addresses
	newConn, raddr, err := MakeHole(ctx, listenConn, resp, key)
	if err != nil {
		return nil, nil, fmt.Errorf("xudp make hole error: %v", err)
	}

	xl.Infof("xudp hole punch successful, remote: %s", raddr.String())
	return newConn, raddr, nil
}

// predictNextPorts generates predicted port addresses based on observed delta.
func predictNextPorts(addrs []string, delta int, count int) []string {
	predicted := make([]string, 0, len(addrs)*count)
	for _, addr := range addrs {
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			continue
		}
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 {
			continue
		}
		for i := 1; i <= count; i++ {
			nextPort := port + delta*i
			if nextPort > 0 && nextPort <= 65535 {
				predicted = append(predicted, net.JoinHostPort(host, strconv.Itoa(nextPort)))
			}
		}
	}
	return predicted
}

// dedup removes duplicate strings while preserving order.
func dedup(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	result := make([]string, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}
