// Copyright 2026 The frp Authors
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

package text

import (
	"math"
	"runtime"
	"runtime/metrics"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// TestMemStatsMapping pins the runtime/metrics to MemStats mapping field by
// field. The sample values are made up so that every combination shows up: tiny
// allocations are added to both Mallocs and Frees, and the combined fields
// (HeapInuse, HeapIdle, HeapSys, StackSys, MSpanSys, MCacheSys) add their parts.
func TestMemStatsMapping(t *testing.T) {
	samples := map[string]uint64{
		rmGCHeapTinyAllocsObjects:               3,
		rmGCHeapAllocsObjects:                   100,
		rmGCHeapFreesObjects:                    40,
		rmGCHeapAllocsBytes:                     1 << 20,
		rmGCHeapObjects:                         900,
		rmGCHeapGoalBytes:                       4 << 20,
		rmMemoryClassesTotalBytes:               50 << 20,
		rmMemoryClassesHeapObjectsBytes:         2 << 20,
		rmMemoryClassesHeapUnusedBytes:          3 << 20,
		rmMemoryClassesHeapReleasedBytes:        5 << 20,
		rmMemoryClassesHeapFreeBytes:            7 << 20,
		rmMemoryClassesHeapStacksBytes:          1 << 20,
		rmMemoryClassesOSStacksBytes:            2 << 20,
		rmMemoryClassesMetadataMSpanInuseBytes:  1 << 19,
		rmMemoryClassesMetadataMSpanFreeBytes:   1 << 18,
		rmMemoryClassesMetadataMCacheInuseBytes: 1 << 17,
		rmMemoryClassesMetadataMCacheFreeBytes:  1 << 16,
		rmMemoryClassesProfilingBucketsBytes:    1 << 15,
		rmMemoryClassesMetadataOtherBytes:       1 << 14,
		rmMemoryClassesOtherBytes:               1 << 13,
	}
	lookup := func(name string) uint64 { return samples[name] }

	var ms runtime.MemStats
	memStatsFromRM(&ms, lookup)

	// Expectations are keyed by family name, so this also checks that each family
	// reads the field the mapping fills.
	want := map[string]float64{
		"go_memstats_alloc_bytes":         2 << 20,
		"go_memstats_alloc_bytes_total":   1 << 20,
		"go_memstats_buck_hash_sys_bytes": 1 << 15,
		"go_memstats_frees_total":         43, // 40 frees + 3 tiny allocs
		"go_memstats_gc_sys_bytes":        1 << 14,
		"go_memstats_heap_alloc_bytes":    2 << 20,
		"go_memstats_heap_idle_bytes":     12 << 20, // released + free
		"go_memstats_heap_inuse_bytes":    5 << 20,  // objects + unused
		"go_memstats_heap_objects":        900,
		"go_memstats_heap_released_bytes": 5 << 20,
		"go_memstats_heap_sys_bytes":      17 << 20, // inuse + idle
		"go_memstats_lookups_total":       0,
		"go_memstats_mallocs_total":       103, // 100 mallocs + 3 tiny allocs
		"go_memstats_mcache_inuse_bytes":  1 << 17,
		"go_memstats_mcache_sys_bytes":    3 << 16, // inuse + free
		"go_memstats_mspan_inuse_bytes":   1 << 19,
		"go_memstats_mspan_sys_bytes":     3 << 18, // inuse + free
		"go_memstats_next_gc_bytes":       4 << 20,
		"go_memstats_other_sys_bytes":     1 << 13,
		"go_memstats_stack_inuse_bytes":   1 << 20,
		"go_memstats_stack_sys_bytes":     3 << 20, // inuse + os-stacks
		"go_memstats_sys_bytes":           50 << 20,
	}
	require.Len(t, want, len(memstatsFamilies), "every family needs an expectation")
	for _, m := range memstatsFamilies {
		expected, ok := want[m.name]
		require.True(t, ok, "%s has no expectation", m.name)
		require.Equal(t, expected, m.get(&ms), "%s", m.name)
	}
}

// TestMemorySamplesMissingName checks that a runtime which does not expose a
// name reads it as zero instead of failing, the way client_golang behaves.
func TestMemorySamplesMissingName(t *testing.T) {
	s := &memorySamples{
		samples: []metrics.Sample{{Name: rmGCHeapObjects}},
		index:   map[string]int{rmGCHeapObjects: 0},
	}
	require.Equal(t, uint64(0), s.value(rmMemoryClassesTotalBytes))
}

// TestMemStatsSameOrderAsClientGolang compares what frpc reports with what
// client_golang reads from the same process at about the same time. The check is
// deliberately loose: the two collectors sample the runtime one after the other,
// so counters move by whatever the test itself allocates in between, and an exact
// comparison would flake.
func TestMemStatsSameOrderAsClientGolang(t *testing.T) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	mfs, err := reg.Gather()
	require.NoError(t, err)

	ours := parseFamilies(t, gather(t))
	checked := 0
	for _, mf := range mfs {
		name := mf.GetName()
		if !strings.HasPrefix(name, "go_memstats_") {
			continue
		}
		if name == "go_memstats_last_gc_time_seconds" {
			// A collection that starts between the two reads jumps this value from
			// zero to the current time; that is not a difference in behavior.
			continue
		}
		want, ok := ours[name]
		require.True(t, ok, "%s is missing", name)
		reference := metricValue(t, mf.GetMetric()[0])
		got := metricValue(t, want.GetMetric()[0])
		require.True(t, sameOrder(reference, got),
			"%s differs too much: ours %v, client_golang %v", name, got, reference)
		checked++
	}
	require.Greater(t, checked, 15, "comparison covered too few families")
}

func metricValue(t *testing.T, m *dto.Metric) float64 {
	t.Helper()
	if g := m.GetGauge(); g != nil {
		return g.GetValue()
	}
	if c := m.GetCounter(); c != nil {
		return c.GetValue()
	}
	t.Fatalf("unexpected metric type for %s", m.String())
	return 0
}

// sameOrder reports whether two readings of the same quantity are close enough
// that they cannot be different quantities: equal, within 64 KiB (drift while
// the test allocates), or within a factor of ten.
func sameOrder(a, b float64) bool {
	if a == b {
		return true
	}
	if math.Abs(a-b) <= 1<<16 {
		return true
	}
	if a <= 0 || b <= 0 {
		return false
	}
	ratio := a / b
	return ratio > 0.1 && ratio < 10
}

// TestConcurrentScrapes covers the shared runtime/metrics sample buffer: scrapes
// run on the listener's goroutines and must not race. Run with -race to check it.
func TestConcurrentScrapes(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterRuntime()

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				var b strings.Builder
				if err := reg.WritePrometheus(&b); err != nil {
					t.Errorf("scrape failed: %v", err)
					return
				}
				if !strings.Contains(b.String(), "go_memstats_heap_objects") {
					t.Error("scrape is missing go_memstats_heap_objects")
					return
				}
			}
		})
	}
	wg.Wait()
}
