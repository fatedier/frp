package ports

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"

	"github.com/fatedier/frp/pkg/config/types"
)

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

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}
