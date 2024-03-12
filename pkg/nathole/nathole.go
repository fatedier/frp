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

package nathole

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fatedier/golib/pool"
	"golang.org/x/net/ipv4"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/xlog"
)

var (
	// mode 0: simple detect mode, usually for both EasyNAT or HardNAT & EasyNAT(Public Network)
	// a. receiver sends detect message with low TTL
	// b. sender sends normal detect message to receiver
	// c. receiver receives detect message and sends back a message to sender
	//
	// mode 1: For HardNAT & EasyNAT, send detect messages to multiple guessed ports.
	// Usually applicable to scenarios where port changes are regular.
	// Most of the steps are the same as mode 0, but EasyNAT is fixed as the receiver and will send detect messages
	// with low TTL to multiple guessed ports of the sender.
	//
	// mode 2: For HardNAT & EasyNAT, ports changes are not regular.
	// a. HardNAT machine will listen on multiple ports and send detect messages with low TTL to EasyNAT machine
	// b. EasyNAT machine will send detect messages to random ports of HardNAT machine.
	//
	// mode 3: For HardNAT & HardNAT, both changes in the ports are regular.
	// Most of the steps are the same as mode 1, but the sender also needs to send detect messages to multiple guessed
	// ports of the receiver.
	//
	// mode 4: For HardNAT & HardNAT, one of the changes in the ports is regular.
	// Regular port changes are usually on the sender side.
	// a. Receiver listens on multiple ports and sends detect messages with low TTL to the sender's guessed range ports.
	// b. Sender sends detect messages to random ports of the receiver.
	SupportedModes = []int{DetectMode0, DetectMode1, DetectMode2, DetectMode3, DetectMode4}
	SupportedRoles = []string{DetectRoleSender, DetectRoleReceiver}

	DetectMode0        = 0
	DetectMode1        = 1
	DetectMode2        = 2
	DetectMode3        = 3
	DetectMode4        = 4
	DetectRoleSender   = "sender"
	DetectRoleReceiver = "receiver"
)

type PrepareResult struct {
	Addrs         []string
	AssistedAddrs []string
	ListenConn    *net.UDPConn
	NatType       string
	Behavior      string
}

// PreCheck is used to check if the proxy is ready for penetration.
// Call this function before calling Prepare to avoid unnecessary preparation work.
func PreCheck(
	ctx context.Context, transporter transport.MessageTransporter,
	proxyName string, timeout time.Duration,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var natHoleRespMsg *msg.NatHoleResp
	transactionID := NewTransactionID()
	m, err := transporter.Do(timeoutCtx, &msg.NatHoleVisitor{
		TransactionID: transactionID,
		ProxyName:     proxyName,
		PreCheck:      true,
	}, transactionID, msg.TypeNameNatHoleResp)
	if err != nil {
		return fmt.Errorf("get natHoleRespMsg error: %v", err)
	}
	mm, ok := m.(*msg.NatHoleResp)
	if !ok {
		return fmt.Errorf("get natHoleRespMsg error: invalid message type")
	}
	natHoleRespMsg = mm

	if natHoleRespMsg.Error != "" {
		return fmt.Errorf("%s", natHoleRespMsg.Error)
	}
	return nil
}

// Prepare is used to do some preparation work before penetration.
func Prepare(stunServers []string) (*PrepareResult, error) {
	// discover for Nat type
	addrs, localAddr, err := Discover(stunServers, "")
	if err != nil {
		return nil, fmt.Errorf("discover error: %v", err)
	}
	if len(addrs) < 2 {
		return nil, fmt.Errorf("discover error: not enough addresses")
	}

	localIPs, _ := ListLocalIPsForNatHole(10)
	natFeature, err := ClassifyNATFeature(addrs, localIPs)
	if err != nil {
		return nil, fmt.Errorf("classify nat feature error: %v", err)
	}

	laddr, err := net.ResolveUDPAddr("udp4", localAddr.String())
	if err != nil {
		return nil, fmt.Errorf("resolve local udp addr error: %v", err)
	}
	listenConn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, fmt.Errorf("listen local udp addr error: %v", err)
	}

	assistedAddrs := make([]string, 0, len(localIPs))
	for _, ip := range localIPs {
		assistedAddrs = append(assistedAddrs, net.JoinHostPort(ip, strconv.Itoa(laddr.Port)))
	}
	return &PrepareResult{
		Addrs:         addrs,
		AssistedAddrs: assistedAddrs,
		ListenConn:    listenConn,
		NatType:       natFeature.NatType,
		Behavior:      natFeature.Behavior,
	}, nil
}

// ExchangeInfo is used to exchange information between client and visitor.
// 1. Send input message to server by msgTransporter.
// 2. Server will gather information from client and visitor and analyze it. Then send back a NatHoleResp message to them to tell them how to do next.
// 3. Receive NatHoleResp message from server.
func ExchangeInfo(
	ctx context.Context, transporter transport.MessageTransporter,
	laneKey string, m msg.Message, timeout time.Duration,
) (*msg.NatHoleResp, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var natHoleRespMsg *msg.NatHoleResp
	m, err := transporter.Do(timeoutCtx, m, laneKey, msg.TypeNameNatHoleResp)
	if err != nil {
		return nil, fmt.Errorf("get natHoleRespMsg error: %v", err)
	}
	mm, ok := m.(*msg.NatHoleResp)
	if !ok {
		return nil, fmt.Errorf("get natHoleRespMsg error: invalid message type")
	}
	natHoleRespMsg = mm

	if natHoleRespMsg.Error != "" {
		return nil, fmt.Errorf("natHoleRespMsg get error info: %s", natHoleRespMsg.Error)
	}
	if len(natHoleRespMsg.CandidateAddrs) == 0 {
		return nil, fmt.Errorf("natHoleRespMsg get empty candidate addresses")
	}
	return natHoleRespMsg, nil
}

// MakeHole is used to make a NAT hole between client and visitor.
func MakeHole(ctx context.Context, listenConn *net.UDPConn, m *msg.NatHoleResp, key []byte) (*net.UDPConn, *net.UDPAddr, error) {
	xl := xlog.FromContextSafe(ctx)
	transactionID := NewTransactionID()
	sendToRangePortsFunc := func(conn *net.UDPConn, addr string) error {
		return sendSidMessage(ctx, conn, m.Sid, transactionID, addr, key, m.DetectBehavior.TTL)
	}

	listenConns := []*net.UDPConn{listenConn}
	var detectAddrs []string
	if m.DetectBehavior.Role == DetectRoleSender {
		// sender
		if m.DetectBehavior.SendDelayMs > 0 {
			time.Sleep(time.Duration(m.DetectBehavior.SendDelayMs) * time.Millisecond)
		}
		detectAddrs = m.AssistedAddrs
		detectAddrs = append(detectAddrs, m.CandidateAddrs...)
	} else {
		// receiver
		if len(m.DetectBehavior.CandidatePorts) == 0 {
			detectAddrs = m.CandidateAddrs
		}

		if m.DetectBehavior.ListenRandomPorts > 0 {
			for i := 0; i < m.DetectBehavior.ListenRandomPorts; i++ {
				tmpConn, err := net.ListenUDP("udp4", nil)
				if err != nil {
					xl.Warnf("listen random udp addr error: %v", err)
					continue
				}
				listenConns = append(listenConns, tmpConn)
			}
		}
	}

	detectAddrs = slices.Compact(detectAddrs)
	for _, detectAddr := range detectAddrs {
		for _, conn := range listenConns {
			if err := sendSidMessage(ctx, conn, m.Sid, transactionID, detectAddr, key, m.DetectBehavior.TTL); err != nil {
				xl.Tracef("send sid message from %s to %s error: %v", conn.LocalAddr(), detectAddr, err)
			}
		}
	}
	if len(m.DetectBehavior.CandidatePorts) > 0 {
		for _, conn := range listenConns {
			sendSidMessageToRangePorts(ctx, conn, m.CandidateAddrs, m.DetectBehavior.CandidatePorts, sendToRangePortsFunc)
		}
	}
	if m.DetectBehavior.SendRandomPorts > 0 {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		for i := range listenConns {
			go sendSidMessageToRandomPorts(ctx, listenConns[i], m.CandidateAddrs, m.DetectBehavior.SendRandomPorts, sendToRangePortsFunc)
		}
	}

	timeout := 5 * time.Second
	if m.DetectBehavior.ReadTimeoutMs > 0 {
		timeout = time.Duration(m.DetectBehavior.ReadTimeoutMs) * time.Millisecond
	}

	if len(listenConns) == 1 {
		raddr, err := waitDetectMessage(ctx, listenConns[0], m.Sid, key, timeout, m.DetectBehavior.Role)
		if err != nil {
			return nil, nil, fmt.Errorf("wait detect message error: %v", err)
		}
		return listenConns[0], raddr, nil
	}

	type result struct {
		lConn *net.UDPConn
		raddr *net.UDPAddr
	}
	resultCh := make(chan result)
	for _, conn := range listenConns {
		go func(lConn *net.UDPConn) {
			addr, err := waitDetectMessage(ctx, lConn, m.Sid, key, timeout, m.DetectBehavior.Role)
			if err != nil {
				lConn.Close()
				return
			}
			select {
			case resultCh <- result{lConn: lConn, raddr: addr}:
			default:
				lConn.Close()
			}
		}(conn)
	}

	select {
	case result := <-resultCh:
		return result.lConn, result.raddr, nil
	case <-time.After(timeout):
		return nil, nil, fmt.Errorf("wait detect message timeout")
	case <-ctx.Done():
		return nil, nil, fmt.Errorf("wait detect message canceled")
	}
}

func waitDetectMessage(
	ctx context.Context, conn *net.UDPConn, sid string, key []byte,
	timeout time.Duration, role string,
) (*net.UDPAddr, error) {
	xl := xlog.FromContextSafe(ctx)
	for {
		buf := pool.GetBuf(1024)
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		n, raddr, err := conn.ReadFromUDP(buf)
		_ = conn.SetReadDeadline(time.Time{})
		if err != nil {
			return nil, err
		}
		xl.Debugf("get udp message local %s, from %s", conn.LocalAddr(), raddr)
		var m msg.NatHoleSid
		if err := DecodeMessageInto(buf[:n], key, &m); err != nil {
			xl.Warnf("decode sid message error: %v", err)
			continue
		}
		pool.PutBuf(buf)

		if m.Sid != sid {
			xl.Warnf("get sid message with wrong sid: %s, expect: %s", m.Sid, sid)
			continue
		}

		if !m.Response {
			// only wait for response messages if we are a sender
			if role == DetectRoleSender {
				continue
			}

			m.Response = true
			buf2, err := EncodeMessage(&m, key)
			if err != nil {
				xl.Warnf("encode sid message error: %v", err)
				continue
			}
			_, _ = conn.WriteToUDP(buf2, raddr)
		}
		return raddr, nil
	}
}

func sendSidMessage(
	ctx context.Context, conn *net.UDPConn,
	sid string, transactionID string, addr string, key []byte, ttl int,
) error {
	xl := xlog.FromContextSafe(ctx)
	ttlStr := ""
	if ttl > 0 {
		ttlStr = fmt.Sprintf(" with ttl %d", ttl)
	}
	xl.Tracef("send sid message from %s to %s%s", conn.LocalAddr(), addr, ttlStr)
	raddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return err
	}
	if transactionID == "" {
		transactionID = NewTransactionID()
	}
	m := &msg.NatHoleSid{
		TransactionID: transactionID,
		Sid:           sid,
		Response:      false,
		Nonce:         strings.Repeat("0", rand.Intn(20)),
	}
	buf, err := EncodeMessage(m, key)
	if err != nil {
		return err
	}
	if ttl > 0 {
		uConn := ipv4.NewConn(conn)
		original, err := uConn.TTL()
		if err != nil {
			xl.Tracef("get ttl error %v", err)
			return err
		}
		xl.Tracef("original ttl %d", original)

		err = uConn.SetTTL(ttl)
		if err != nil {
			xl.Tracef("set ttl error %v", err)
		} else {
			defer func() {
				_ = uConn.SetTTL(original)
			}()
		}
	}

	if _, err := conn.WriteToUDP(buf, raddr); err != nil {
		return err
	}
	return nil
}

func sendSidMessageToRangePorts(
	ctx context.Context, conn *net.UDPConn, addrs []string, ports []msg.PortsRange,
	sendFunc func(*net.UDPConn, string) error,
) {
	xl := xlog.FromContextSafe(ctx)
	for _, ip := range slices.Compact(parseIPs(addrs)) {
		for _, portsRange := range ports {
			for i := portsRange.From; i <= portsRange.To; i++ {
				detectAddr := net.JoinHostPort(ip, strconv.Itoa(i))
				if err := sendFunc(conn, detectAddr); err != nil {
					xl.Tracef("send sid message from %s to %s error: %v", conn.LocalAddr(), detectAddr, err)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}
	}
}

func sendSidMessageToRandomPorts(
	ctx context.Context, conn *net.UDPConn, addrs []string, count int,
	sendFunc func(*net.UDPConn, string) error,
) {
	xl := xlog.FromContextSafe(ctx)
	used := sets.New[int]()
	getUnusedPort := func() int {
		for i := 0; i < 10; i++ {
			port := rand.Intn(65535-1024) + 1024
			if !used.Has(port) {
				used.Insert(port)
				return port
			}
		}
		return 0
	}

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		port := getUnusedPort()
		if port == 0 {
			continue
		}

		for _, ip := range slices.Compact(parseIPs(addrs)) {
			detectAddr := net.JoinHostPort(ip, strconv.Itoa(port))
			if err := sendFunc(conn, detectAddr); err != nil {
				xl.Tracef("send sid message from %s to %s error: %v", conn.LocalAddr(), detectAddr, err)
			}
			time.Sleep(time.Millisecond * 15)
		}
	}
}

func parseIPs(addrs []string) []string {
	var ips []string
	for _, addr := range addrs {
		if ip, _, err := net.SplitHostPort(addr); err == nil {
			ips = append(ips, ip)
		}
	}
	return ips
}
