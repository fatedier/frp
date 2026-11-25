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

// Package text writes Prometheus metrics in the text exposition format.
//
// frp uses it for frpc, which only exposes Go runtime metrics. Implementing
// that subset here keeps client_golang and its protobuf dependency out of the
// frpc binary; frps keeps using client_golang, which also gives it the
// business metrics and the HTTP handler instrumentation.
package text

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ContentType is the content type of the text exposition format.
const ContentType = "text/plain; version=0.0.4; charset=utf-8"

// metricType is the Prometheus metric type.
type metricType string

const (
	typeCounter metricType = "counter"
	typeGauge   metricType = "gauge"
	typeSummary metricType = "summary"
)

type label struct {
	name  string
	value string
}

// sample is one time series. name is empty for plain families, where the
// family name is used; summaries set it to carry the _sum and _count suffixes.
type sample struct {
	name   string
	labels []label
	value  float64
}

// family is one metric family ready to be encoded.
type family struct {
	name    string
	help    string
	typ     metricType
	samples []sample
}

// encode writes families in the order they are given.
func encode(w io.Writer, families []family) error {
	for _, f := range families {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n", f.name, escapeHelp(f.help)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "# TYPE %s %s\n", f.name, f.typ); err != nil {
			return err
		}
		for _, s := range f.samples {
			if err := encodeSample(w, f.name, s); err != nil {
				return err
			}
		}
	}
	return nil
}

func encodeSample(w io.Writer, familyName string, s sample) error {
	name := s.name
	if name == "" {
		name = familyName
	}
	if _, err := io.WriteString(w, name); err != nil {
		return err
	}
	if len(s.labels) > 0 {
		if err := encodeLabels(w, s.labels); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, " %s\n", formatValue(s.value))
	return err
}

func encodeLabels(w io.Writer, labels []label) error {
	if _, err := io.WriteString(w, "{"); err != nil {
		return err
	}
	for i, l := range labels {
		if i > 0 {
			if _, err := io.WriteString(w, ","); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, `%s="%s"`, l.name, escapeLabelValue(l.value)); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "}")
	return err
}

// escapeLabelValue applies the three escapes the text format defines for
// label values.
func escapeLabelValue(v string) string {
	if !strings.ContainsAny(v, "\\\"\n") {
		return v
	}
	var b strings.Builder
	b.Grow(len(v) + 8)
	for _, r := range v {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeHelp applies the two escapes the text format defines for help text.
func escapeHelp(v string) string {
	if !strings.ContainsAny(v, "\\\n") {
		return v
	}
	var b strings.Builder
	b.Grow(len(v) + 8)
	for _, r := range v {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatValue renders a float the way the Prometheus text format does, so a
// sample reads the same whichever implementation produced it.
func formatValue(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
