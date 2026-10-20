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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
)

// gather returns what frpc would serve.
func gather(t *testing.T) string {
	t.Helper()
	reg := NewRegistry()
	reg.RegisterRuntime()
	var b strings.Builder
	require.NoError(t, reg.WritePrometheus(&b))
	return b.String()
}

// parseFamilies decodes the text exposition format with the Prometheus parser,
// so anything a scraper would reject fails the test.
func parseFamilies(t *testing.T, body string) map[string]*dto.MetricFamily {
	t.Helper()
	parser := expfmt.TextParser{}
	families, err := parser.TextToMetricFamilies(strings.NewReader(body))
	require.NoError(t, err, "output must be valid text exposition format")
	return families
}

// TestMatchesClientGolang checks that every go_* family this package emits has
// the same name, type and help text as client_golang's default Go collector. The
// official collector is used as the oracle, so a difference shows up here rather
// than in someone's dashboard.
func TestMatchesClientGolang(t *testing.T) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	mfs, err := reg.Gather()
	require.NoError(t, err)

	reference := make(map[string]*dto.MetricFamily, len(mfs))
	for _, mf := range mfs {
		reference[mf.GetName()] = mf
	}
	ours := parseFamilies(t, gather(t))

	checked := 0
	for name, want := range reference {
		if !strings.HasPrefix(name, "go_") {
			continue
		}
		got, ok := ours[name]
		require.True(t, ok, "%s is missing", name)
		require.Equal(t, want.GetType(), got.GetType(), "%s type", name)
		require.Equal(t, want.GetHelp(), got.GetHelp(), "%s help", name)
		checked++
	}
	require.Greater(t, checked, 20, "comparison covered too few families")
}

// TestNoBusinessMetrics pins what frpc must not expose: no frp_ business metrics,
// none of the handler instrumentation client_golang adds, and no process_
// families. frpc reports Go runtime metrics only, on every platform.
func TestNoBusinessMetrics(t *testing.T) {
	body := gather(t)
	require.NotContains(t, body, "frp_")
	require.NotContains(t, body, "promhttp_")
	require.NotContains(t, body, "process_")
}

func TestHandler(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterRuntime()
	rec := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, ContentType, rec.Header().Get("Content-Type"))

	families := parseFamilies(t, rec.Body.String())
	require.Contains(t, families, "go_goroutines")
	gc, ok := families["go_gc_duration_seconds"]
	require.True(t, ok)
	require.Equal(t, dto.MetricType_SUMMARY, gc.GetType())
	require.Len(t, gc.GetMetric()[0].GetSummary().GetQuantile(), 5)
}

func TestEmptyRegistry(t *testing.T) {
	var b strings.Builder
	require.NoError(t, NewRegistry().WritePrometheus(&b))
	require.Empty(t, b.String())
}

func TestEscaping(t *testing.T) {
	var b strings.Builder
	require.NoError(t, encode(&b, []family{{
		name:    "test_labels",
		help:    "help with \\ and\nnewline",
		typ:     typeGauge,
		samples: []sample{{labels: []label{{name: "v", value: "a\"b\\c\nd"}}, value: 1}},
	}}))
	require.Equal(t, "# HELP test_labels help with \\\\ and\\nnewline\n"+
		"# TYPE test_labels gauge\n"+
		"test_labels{v=\"a\\\"b\\\\c\\nd\"} 1\n", b.String())
}
