package wait

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"
)

func TestFastBackoffManagerUsesClock(t *testing.T) {
	require := require.New(t)

	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start)
	backoff := newFastBackoffManagerWithClock(FastBackoffOptions{
		Duration: time.Second,
	}, clk).(*fastBackoffImpl)

	require.Equal(time.Second, backoff.Backoff(0, false))
	require.Equal(start, backoff.lastCalledTime)

	next := start.Add(time.Minute)
	clk.SetTime(next)
	require.Equal(time.Second, backoff.Backoff(time.Second, false))
	require.Equal(next, backoff.lastCalledTime)
}
