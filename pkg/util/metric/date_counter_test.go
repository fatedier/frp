package metric

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"
)

func TestDateCounter(t *testing.T) {
	require := require.New(t)

	dc := NewDateCounter(3)
	dc.Inc(10)
	require.EqualValues(10, dc.TodayCount())

	dc.Dec(5)
	require.EqualValues(5, dc.TodayCount())

	counts := dc.GetLastDaysCount(3)
	require.EqualValues(3, len(counts))
	require.EqualValues(5, counts[0])
	require.EqualValues(0, counts[1])
	require.EqualValues(0, counts[2])

	dcTmp := dc.Snapshot()
	require.EqualValues(5, dcTmp.TodayCount())
}

func TestDateCounterRotate(t *testing.T) {
	loc := time.FixedZone("test", 8*60*60)
	lastUpdateDate := time.Date(2026, time.May, 8, 0, 0, 0, 0, loc)

	tests := []struct {
		name               string
		now                time.Time
		want               []int64
		wantLastUpdateDate time.Time
	}{
		{
			name:               "same day",
			now:                time.Date(2026, time.May, 8, 12, 30, 0, 0, loc),
			want:               []int64{10, 7, 3},
			wantLastUpdateDate: lastUpdateDate,
		},
		{
			name:               "clock skew",
			now:                time.Date(2026, time.May, 7, 12, 30, 0, 0, loc),
			want:               []int64{10, 7, 3},
			wantLastUpdateDate: lastUpdateDate,
		},
		{
			name:               "one day",
			now:                time.Date(2026, time.May, 9, 12, 30, 0, 0, loc),
			want:               []int64{0, 10, 7},
			wantLastUpdateDate: time.Date(2026, time.May, 9, 0, 0, 0, 0, loc),
		},
		{
			name:               "two days",
			now:                time.Date(2026, time.May, 10, 12, 30, 0, 0, loc),
			want:               []int64{0, 0, 10},
			wantLastUpdateDate: time.Date(2026, time.May, 10, 0, 0, 0, 0, loc),
		},
		{
			name:               "all reserved days elapsed",
			now:                time.Date(2026, time.May, 11, 12, 30, 0, 0, loc),
			want:               []int64{0, 0, 0},
			wantLastUpdateDate: time.Date(2026, time.May, 11, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			dc := newStandardDateCounter(3)
			dc.counts = []int64{10, 7, 3}
			dc.lastUpdateDate = lastUpdateDate

			dc.mu.Lock()
			dc.rotate(tt.now)
			dc.mu.Unlock()

			require.Equal(tt.want, dc.counts)
			require.Equal(tt.wantLastUpdateDate, dc.lastUpdateDate)
		})
	}
}

func TestDateCounterGetLastDaysCountReturnsCopy(t *testing.T) {
	require := require.New(t)

	clk := clocktesting.NewFakeClock(time.Date(2026, time.May, 8, 12, 30, 0, 0, time.Local))
	dc := newStandardDateCounterWithClock(3, clk)
	dc.counts = []int64{10, 7, 3}

	counts := dc.GetLastDaysCount(2)
	require.Equal([]int64{10, 7}, counts)

	counts[0] = 100
	require.Equal([]int64{10, 7}, dc.GetLastDaysCount(2))
}

func TestDateCounterClear(t *testing.T) {
	require := require.New(t)

	dc := newStandardDateCounter(3)
	dc.counts = []int64{10, 7, 3}

	dc.Clear()

	require.Equal([]int64{0, 0, 0}, dc.counts)
}

func TestDateCounterConcurrentAccess(t *testing.T) {
	clk := clocktesting.NewFakeClock(time.Date(2026, time.May, 8, 12, 30, 0, 0, time.Local))
	dc := newStandardDateCounterWithClock(3, clk)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				dc.Inc(1)
				dc.Dec(1)
				_ = dc.TodayCount()
				_ = dc.GetLastDaysCount(3)
				_ = dc.Snapshot()
			}
		})
	}
	wg.Wait()
}
