package nathole

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"
)

func TestAnalyzerUsesClockForRecordTimestamps(t *testing.T) {
	require := require.New(t)

	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start)
	analyzer := newAnalyzerWithClock(time.Hour, clk)
	clientFeature := &NatFeature{NatType: EasyNAT, Behavior: BehaviorNoChange}
	visitorFeature := &NatFeature{NatType: EasyNAT, Behavior: BehaviorNoChange}

	mode, index, _, _ := analyzer.GetRecommandBehaviors("key", clientFeature, visitorFeature)
	require.Equal(start, analyzer.records["key"].lastUpdateTime)

	updatedAt := start.Add(time.Minute)
	clk.SetTime(updatedAt)
	analyzer.ReportSuccess("key", mode, index)
	require.Equal(updatedAt, analyzer.records["key"].lastUpdateTime)

	clk.SetTime(start.Add(2 * time.Hour))
	count, total := analyzer.Clean()
	require.Equal(1, count)
	require.Equal(1, total)
	require.Empty(analyzer.records)
}
