package mem

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"
)

func TestServerMetricsUsesClockForProxyTimestamps(t *testing.T) {
	require := require.New(t)

	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start)
	metrics := newServerMetricsWithClock(clk)

	metrics.NewProxy("proxy", "tcp", "user", "client-id")
	require.Equal(start, metrics.info.ProxyStatistics["proxy"].LastStartTime)

	closedAt := start.Add(time.Minute)
	clk.SetTime(closedAt)
	metrics.CloseProxy("proxy", "tcp")
	require.Equal(closedAt, metrics.info.ProxyStatistics["proxy"].LastCloseTime)

	stats := metrics.GetProxyByName("proxy")
	require.Equal(start.Format("01-02 15:04:05"), stats.LastStartTime)
	require.Equal(closedAt.Format("01-02 15:04:05"), stats.LastCloseTime)
	require.Equal(start.Unix(), stats.LastStartAt)
	require.Equal(closedAt.Unix(), stats.LastCloseAt)
}

func TestServerMetricsClearUselessInfoUsesClock(t *testing.T) {
	require := require.New(t)

	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start.Add(25 * time.Hour))
	metrics := newServerMetricsWithClock(clk)
	metrics.info.ProxyStatistics["proxy"] = &ProxyStatistics{
		Name:          "proxy",
		LastStartTime: start.Add(-time.Hour),
		LastCloseTime: start,
	}

	count, total := metrics.clearUselessInfo(24 * time.Hour)

	require.Equal(1, count)
	require.Equal(1, total)
	require.Empty(metrics.info.ProxyStatistics)
}

func TestServerMetricsClearOfflineProxiesPreservesLegacyTotal(t *testing.T) {
	require := require.New(t)

	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start.Add(time.Minute))
	metrics := newServerMetricsWithClock(clk)
	metrics.info.ProxyStatistics["offline"] = &ProxyStatistics{
		Name:          "offline",
		LastStartTime: start.Add(-time.Hour),
		LastCloseTime: start,
	}
	metrics.info.ProxyStatistics["online"] = &ProxyStatistics{
		Name:          "online",
		LastStartTime: start,
	}

	cleared, total := metrics.ClearOfflineProxies()

	require.Equal(1, cleared)
	require.Equal(2, total)
	require.False(metrics.hasProxyStatistics("offline"))
	require.True(metrics.hasProxyStatistics("online"))
}

func TestServerMetricsPruneOfflineProxiesReportsTotalStats(t *testing.T) {
	require := require.New(t)

	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start.Add(time.Minute))
	metrics := newServerMetricsWithClock(clk)
	metrics.info.ProxyStatistics["offline"] = &ProxyStatistics{
		Name:          "offline",
		LastStartTime: start.Add(-time.Hour),
		LastCloseTime: start,
	}
	metrics.info.ProxyStatistics["online"] = &ProxyStatistics{
		Name:          "online",
		LastStartTime: start,
	}
	metrics.info.ProxyStatistics["restarted"] = &ProxyStatistics{
		Name:          "restarted",
		LastStartTime: start.Add(30 * time.Second),
		LastCloseTime: start,
	}
	metrics.info.ProxyStatistics["same-time"] = &ProxyStatistics{
		Name:          "same-time",
		LastStartTime: start,
		LastCloseTime: start,
	}

	cleared, total := metrics.PruneOfflineProxies()

	require.Equal(1, cleared)
	require.Equal(4, total)
	require.False(metrics.hasProxyStatistics("offline"))
	require.True(metrics.hasProxyStatistics("online"))
	require.True(metrics.hasProxyStatistics("restarted"))
	require.True(metrics.hasProxyStatistics("same-time"))

	cleared, total = metrics.PruneOfflineProxies()
	require.Equal(0, cleared)
	require.Equal(3, total)
}

func TestServerMetricsRunUsesClockTicker(t *testing.T) {
	require := require.New(t)

	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start)
	metrics := newServerMetricsWithClock(clk)
	metrics.info.ProxyStatistics["proxy"] = &ProxyStatistics{
		Name:          "proxy",
		LastStartTime: start.Add(-time.Hour),
		LastCloseTime: start,
	}

	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		metrics.runUntil(stopCh)
	}()
	t.Cleanup(func() {
		close(stopCh)
		<-done
	})

	require.Eventually(clk.HasWaiters, time.Second, time.Millisecond)
	clk.Step(8 * 24 * time.Hour)

	require.Eventually(func() bool {
		return !metrics.hasProxyStatistics("proxy")
	}, time.Second, time.Millisecond)
}

func (m *serverMetrics) hasProxyStatistics(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.info.ProxyStatistics[name]
	return ok
}
