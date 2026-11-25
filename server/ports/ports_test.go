package ports

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"

	"github.com/fatedier/frp/pkg/config/types"
)

func TestManagerDoesNotReuseReservedPortWhileAcquired(t *testing.T) {
	for _, network := range []string{"tcp", "udp"} {
		t.Run(network, func(t *testing.T) {
			port, closePort := listenOnLoopback(t, network)
			defer closePort()
			alternate, closeAlternate := listenOnLoopback(t, network)
			defer closeAlternate()
			require.NoError(t, closePort())
			require.NoError(t, closeAlternate())
			pm := NewManager(network, "127.0.0.1", []types.PortsRange{{Single: port}, {Single: alternate}})

			_, err := pm.Acquire("first", port)
			require.NoError(t, err)
			pm.Release(port)

			_, err = pm.Acquire("other", alternate)
			require.NoError(t, err)
			got, err := pm.Acquire("first", 0)
			require.NoError(t, err)
			require.Equal(t, port, got)
			pm.Release(port)

			_, err = pm.Acquire("second", port)
			require.NoError(t, err)
			// The second proxy owns the port, even before it starts listening.
			_, err = pm.Acquire("first", 0)
			require.ErrorIs(t, err, ErrNoAvailablePort)

			pm.Release(alternate)
			got, err = pm.Acquire("first", 0)
			require.NoError(t, err)
			require.Equal(t, alternate, got)
			_, err = pm.Acquire("third", port)
			require.ErrorIs(t, err, ErrPortAlreadyUsed)
		})
	}
}

func TestManagerUsesClockForPortTimestamps(t *testing.T) {
	require := require.New(t)

	port := freeTCPPort(t)
	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start)
	pm := newManagerWithClock("tcp", "127.0.0.1", []types.PortsRange{{Single: port}}, clk)

	realPort, err := pm.Acquire("proxy", port)
	require.NoError(err)
	require.Equal(port, realPort)
	require.Equal(start, pm.usedPorts[port].UpdateTime)

	releasedAt := start.Add(time.Minute)
	clk.SetTime(releasedAt)
	pm.Release(port)

	require.Equal(releasedAt, pm.reservedPorts["proxy"].UpdateTime)
}

func TestManagerCleanReservedPortsWorkerUsesClockTicker(t *testing.T) {
	require := require.New(t)

	port := freeTCPPort(t)
	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start)
	pm := newManagerWithClock("tcp", "127.0.0.1", []types.PortsRange{{Single: port}}, clk)

	realPort, err := pm.Acquire("proxy", port)
	require.NoError(err)
	require.Equal(port, realPort)
	pm.Release(port)
	require.True(pm.hasReservedPort("proxy"))

	require.Eventually(clk.HasWaiters, time.Second, time.Millisecond)
	clk.Step(MaxPortReservedDuration + CleanReservedPortsInterval + time.Minute)

	require.Eventually(func() bool {
		return !pm.hasReservedPort("proxy")
	}, time.Second, time.Millisecond)
}

func (pm *Manager) hasReservedPort(name string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_, ok := pm.reservedPorts[name]
	return ok
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	port, closePort := listenOnLoopback(t, "tcp")
	require.NoError(t, closePort())
	return port
}

func listenOnLoopback(t *testing.T, network string) (int, func() error) {
	t.Helper()
	if network == "udp" {
		conn, err := net.ListenPacket(network, "127.0.0.1:0")
		require.NoError(t, err)
		return conn.LocalAddr().(*net.UDPAddr).Port, conn.Close
	}

	listener, err := net.Listen(network, "127.0.0.1:0")
	require.NoError(t, err)
	return listener.Addr().(*net.TCPAddr).Port, listener.Close
}
