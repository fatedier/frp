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
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"sync"
	"time"
)

// memstats is one go_memstats_* family. The values are read the same way
// client_golang reads them since Go 1.17: from runtime/metrics, which does not
// stop the world, and then mapped onto runtime.MemStats fields.
type memstats struct {
	name string
	help string
	typ  metricType
	get  func(*runtime.MemStats) float64
}

// memstatsFamilies matches the go_memstats_* families of client_golang's default
// registry, including their help strings and types.
var memstatsFamilies = []memstats{
	{
		name: "go_memstats_alloc_bytes",
		help: "Number of bytes allocated and still in use.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.Alloc) },
	},
	{
		name: "go_memstats_alloc_bytes_total",
		help: "Total number of bytes allocated, even if freed.",
		typ:  typeCounter,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.TotalAlloc) },
	},
	{
		name: "go_memstats_buck_hash_sys_bytes",
		help: "Number of bytes used by the profiling bucket hash table.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.BuckHashSys) },
	},
	{
		name: "go_memstats_frees_total",
		help: "Total number of frees.",
		typ:  typeCounter,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.Frees) },
	},
	{
		name: "go_memstats_gc_sys_bytes",
		help: "Number of bytes used for garbage collection system metadata.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.GCSys) },
	},
	{
		name: "go_memstats_heap_alloc_bytes",
		help: "Number of heap bytes allocated and still in use.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.HeapAlloc) },
	},
	{
		name: "go_memstats_heap_idle_bytes",
		help: "Number of heap bytes waiting to be used.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.HeapIdle) },
	},
	{
		name: "go_memstats_heap_inuse_bytes",
		help: "Number of heap bytes that are in use.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.HeapInuse) },
	},
	{
		name: "go_memstats_heap_objects",
		help: "Number of allocated objects.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.HeapObjects) },
	},
	{
		name: "go_memstats_heap_released_bytes",
		help: "Number of heap bytes released to OS.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.HeapReleased) },
	},
	{
		name: "go_memstats_heap_sys_bytes",
		help: "Number of heap bytes obtained from system.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.HeapSys) },
	},
	{
		name: "go_memstats_lookups_total",
		help: "Total number of pointer lookups.",
		typ:  typeCounter,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.Lookups) },
	},
	{
		name: "go_memstats_mallocs_total",
		help: "Total number of mallocs.",
		typ:  typeCounter,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.Mallocs) },
	},
	{
		name: "go_memstats_mcache_inuse_bytes",
		help: "Number of bytes in use by mcache structures.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.MCacheInuse) },
	},
	{
		name: "go_memstats_mcache_sys_bytes",
		help: "Number of bytes used for mcache structures obtained from system.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.MCacheSys) },
	},
	{
		name: "go_memstats_mspan_inuse_bytes",
		help: "Number of bytes in use by mspan structures.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.MSpanInuse) },
	},
	{
		name: "go_memstats_mspan_sys_bytes",
		help: "Number of bytes used for mspan structures obtained from system.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.MSpanSys) },
	},
	{
		name: "go_memstats_next_gc_bytes",
		help: "Number of heap bytes when next garbage collection will take place.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.NextGC) },
	},
	{
		name: "go_memstats_other_sys_bytes",
		help: "Number of bytes used for other system allocations.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.OtherSys) },
	},
	{
		name: "go_memstats_stack_inuse_bytes",
		help: "Number of bytes in use by the stack allocator.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.StackInuse) },
	},
	{
		name: "go_memstats_stack_sys_bytes",
		help: "Number of bytes obtained from system for stack allocator.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.StackSys) },
	},
	{
		name: "go_memstats_sys_bytes",
		help: "Number of bytes obtained from system.",
		typ:  typeGauge,
		get:  func(ms *runtime.MemStats) float64 { return float64(ms.Sys) },
	},
}

// The runtime/metrics names below are the ones client_golang v1.19.1 reads to
// build the go_memstats_* families. See prometheus/go_collector_latest.go in
// that module (Apache-2.0).
const (
	rmGCHeapTinyAllocsObjects               = "/gc/heap/tiny/allocs:objects"
	rmGCHeapAllocsObjects                   = "/gc/heap/allocs:objects"
	rmGCHeapFreesObjects                    = "/gc/heap/frees:objects"
	rmGCHeapAllocsBytes                     = "/gc/heap/allocs:bytes"
	rmGCHeapObjects                         = "/gc/heap/objects:objects"
	rmGCHeapGoalBytes                       = "/gc/heap/goal:bytes"
	rmMemoryClassesTotalBytes               = "/memory/classes/total:bytes"
	rmMemoryClassesHeapObjectsBytes         = "/memory/classes/heap/objects:bytes"
	rmMemoryClassesHeapUnusedBytes          = "/memory/classes/heap/unused:bytes"
	rmMemoryClassesHeapReleasedBytes        = "/memory/classes/heap/released:bytes"
	rmMemoryClassesHeapFreeBytes            = "/memory/classes/heap/free:bytes"
	rmMemoryClassesHeapStacksBytes          = "/memory/classes/heap/stacks:bytes"
	rmMemoryClassesOSStacksBytes            = "/memory/classes/os-stacks:bytes"
	rmMemoryClassesMetadataMSpanInuseBytes  = "/memory/classes/metadata/mspan/inuse:bytes"
	rmMemoryClassesMetadataMSpanFreeBytes   = "/memory/classes/metadata/mspan/free:bytes"
	rmMemoryClassesMetadataMCacheInuseBytes = "/memory/classes/metadata/mcache/inuse:bytes"
	rmMemoryClassesMetadataMCacheFreeBytes  = "/memory/classes/metadata/mcache/free:bytes"
	rmMemoryClassesProfilingBucketsBytes    = "/memory/classes/profiling/buckets:bytes"
	rmMemoryClassesMetadataOtherBytes       = "/memory/classes/metadata/other:bytes"
	rmMemoryClassesOtherBytes               = "/memory/classes/other:bytes"
)

// memoryNames lists every runtime/metrics name memStatsFromRM reads.
var memoryNames = []string{
	rmGCHeapTinyAllocsObjects,
	rmGCHeapAllocsObjects,
	rmGCHeapFreesObjects,
	rmGCHeapAllocsBytes,
	rmGCHeapObjects,
	rmGCHeapGoalBytes,
	rmMemoryClassesTotalBytes,
	rmMemoryClassesHeapObjectsBytes,
	rmMemoryClassesHeapUnusedBytes,
	rmMemoryClassesHeapReleasedBytes,
	rmMemoryClassesHeapFreeBytes,
	rmMemoryClassesHeapStacksBytes,
	rmMemoryClassesOSStacksBytes,
	rmMemoryClassesMetadataMSpanInuseBytes,
	rmMemoryClassesMetadataMSpanFreeBytes,
	rmMemoryClassesMetadataMCacheInuseBytes,
	rmMemoryClassesMetadataMCacheFreeBytes,
	rmMemoryClassesProfilingBucketsBytes,
	rmMemoryClassesMetadataOtherBytes,
	rmMemoryClassesOtherBytes,
}

// memorySamples holds the sample buffer of one scrape and the index that maps a
// runtime/metrics name to its slot.
type memorySamples struct {
	samples []metrics.Sample
	index   map[string]int
}

// memStatsSnapshot takes the Go memory statistics frpc reports. It reads the
// runtime/metrics package instead of calling runtime.ReadMemStats, which stops
// the world on every call, and memStatsFromRM maps the samples onto the old
// field names.
//
// The sample buffer is reused between scrapes, so concurrent scrapes take turns:
// runtime/metrics.Read writes into the samples it is handed.
func memStatsSnapshot() runtime.MemStats {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	if memoryBuffer == nil {
		memoryBuffer = newMemorySamples()
	}
	// metrics.Read indexes into the slice, so an empty buffer has to be skipped.
	if len(memoryBuffer.samples) > 0 {
		metrics.Read(memoryBuffer.samples)
	}

	var ms runtime.MemStats
	memStatsFromRM(&ms, memoryBuffer.value)
	return ms
}

var (
	memoryMu     sync.Mutex
	memoryBuffer *memorySamples
)

// newMemorySamples builds the sample buffer from the names this runtime supports.
// A name the runtime does not know is left out and reads as zero, which is what
// client_golang does as well.
func newMemorySamples() *memorySamples {
	supported := make(map[string]struct{}, len(memoryNames))
	for _, d := range metrics.All() {
		supported[d.Name] = struct{}{}
	}
	s := &memorySamples{index: make(map[string]int, len(memoryNames))}
	for _, name := range memoryNames {
		if _, ok := supported[name]; !ok {
			continue
		}
		s.index[name] = len(s.samples)
		s.samples = append(s.samples, metrics.Sample{Name: name})
	}
	return s
}

// value returns the sampled value of name, or zero when the runtime does not
// expose it.
func (s *memorySamples) value(name string) uint64 {
	i, ok := s.index[name]
	if !ok {
		return 0
	}
	return s.samples[i].Value.Uint64()
}

// memStatsFromRM maps runtime/metrics samples onto a runtime.MemStats, using the
// same field combinations as client_golang v1.19.1, so the go_memstats_* values
// match the ones the official collector reports for the same process. GCCPUFraction
// is left at zero, as client_golang omits it too.
func memStatsFromRM(ms *runtime.MemStats, lookup func(string) uint64) {
	// MemStats counts tiny allocations in both Mallocs and Frees. client_golang
	// keeps that behavior because Mallocs-Frees is then a live object count.
	tinyAllocs := lookup(rmGCHeapTinyAllocsObjects)
	ms.Mallocs = lookup(rmGCHeapAllocsObjects) + tinyAllocs
	ms.Frees = lookup(rmGCHeapFreesObjects) + tinyAllocs

	ms.TotalAlloc = lookup(rmGCHeapAllocsBytes)
	ms.Sys = lookup(rmMemoryClassesTotalBytes)
	ms.Lookups = 0 // Already always zero.
	ms.HeapAlloc = lookup(rmMemoryClassesHeapObjectsBytes)
	ms.Alloc = ms.HeapAlloc
	ms.HeapInuse = ms.HeapAlloc + lookup(rmMemoryClassesHeapUnusedBytes)
	ms.HeapReleased = lookup(rmMemoryClassesHeapReleasedBytes)
	ms.HeapIdle = ms.HeapReleased + lookup(rmMemoryClassesHeapFreeBytes)
	ms.HeapSys = ms.HeapInuse + ms.HeapIdle
	ms.HeapObjects = lookup(rmGCHeapObjects)
	ms.StackInuse = lookup(rmMemoryClassesHeapStacksBytes)
	ms.StackSys = ms.StackInuse + lookup(rmMemoryClassesOSStacksBytes)
	ms.MSpanInuse = lookup(rmMemoryClassesMetadataMSpanInuseBytes)
	ms.MSpanSys = ms.MSpanInuse + lookup(rmMemoryClassesMetadataMSpanFreeBytes)
	ms.MCacheInuse = lookup(rmMemoryClassesMetadataMCacheInuseBytes)
	ms.MCacheSys = ms.MCacheInuse + lookup(rmMemoryClassesMetadataMCacheFreeBytes)
	ms.BuckHashSys = lookup(rmMemoryClassesProfilingBucketsBytes)
	ms.GCSys = lookup(rmMemoryClassesMetadataOtherBytes)
	ms.OtherSys = lookup(rmMemoryClassesOtherBytes)
	ms.NextGC = lookup(rmGCHeapGoalBytes)
}

// gcQuantileLabels matches the quantiles client_golang reports in the
// go_gc_duration_seconds summary.
var gcQuantileLabels = []string{"0", "0.25", "0.5", "0.75", "1"}

// runtimeFamilies reads the runtime once per scrape rather than once per metric.
func runtimeFamilies() []family {
	ms := memStatsSnapshot()

	out := make([]family, 0, len(memstatsFamilies)+5)
	for _, m := range memstatsFamilies {
		out = append(out, family{
			name:    m.name,
			help:    m.help,
			typ:     m.typ,
			samples: []sample{{value: m.get(&ms)}},
		})
	}

	// debug.ReadGCStats only takes the mheap lock; it does not stop the world.
	var gc debug.GCStats
	gc.PauseQuantiles = make([]time.Duration, len(gcQuantileLabels))
	debug.ReadGCStats(&gc)

	gcSamples := make([]sample, 0, len(gcQuantileLabels)+2)
	for i, q := range gcQuantileLabels {
		var pause time.Duration
		if i < len(gc.PauseQuantiles) {
			pause = gc.PauseQuantiles[i]
		}
		gcSamples = append(gcSamples, sample{
			labels: []label{{name: "quantile", value: q}},
			value:  pause.Seconds(),
		})
	}
	gcSamples = append(gcSamples,
		sample{name: "go_gc_duration_seconds_sum", value: gc.PauseTotal.Seconds()},
		sample{name: "go_gc_duration_seconds_count", value: float64(gc.NumGC)},
	)

	return append(out,
		family{
			name:    "go_gc_duration_seconds",
			help:    "A summary of the pause duration of garbage collection cycles.",
			typ:     typeSummary,
			samples: gcSamples,
		},
		family{
			name:    "go_goroutines",
			help:    "Number of goroutines that currently exist.",
			typ:     typeGauge,
			samples: []sample{{value: float64(runtime.NumGoroutine())}},
		},
		family{
			name:    "go_info",
			help:    "Information about the Go environment.",
			typ:     typeGauge,
			samples: []sample{{labels: []label{{name: "version", value: runtime.Version()}}, value: 1}},
		},
		family{
			name: "go_memstats_last_gc_time_seconds",
			help: "Number of seconds since 1970 of last garbage collection.",
			typ:  typeGauge,
			samples: []sample{{
				value: float64(gc.LastGC.UnixNano()) / float64(time.Second),
			}},
		},
		family{
			name:    "go_threads",
			help:    "Number of OS threads created.",
			typ:     typeGauge,
			samples: []sample{{value: float64(threadCount())}},
		},
	)
}

func threadCount() int {
	n, _ := runtime.ThreadCreateProfile(nil)
	return n
}
