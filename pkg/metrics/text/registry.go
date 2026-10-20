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
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// collector returns the families of one scrape.
type collector func() []family

// familiesPerCollector sizes the gather buffer; the runtime collector is the
// only one today and emits the 22 memstats families plus a handful of others.
const familiesPerCollector = 32

// Registry holds the collectors exposed on one listener.
type Registry struct {
	mu         sync.RWMutex
	collectors []collector
}

// NewRegistry returns a registry with no collectors.
func NewRegistry() *Registry {
	return &Registry{}
}

// RegisterRuntime registers the Go runtime collector.
func (r *Registry) RegisterRuntime() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors = append(r.collectors, runtimeFamilies)
}

// WritePrometheus writes every collected family in the text exposition format.
func (r *Registry) WritePrometheus(w io.Writer) error {
	return encode(w, r.gather())
}

// Handler returns an http.Handler serving the registry.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Encode into memory first so that a failure never sends a truncated body.
		var b strings.Builder
		if err := r.WritePrometheus(&b); err != nil {
			http.Error(w, "metrics unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", ContentType)
		_, _ = io.WriteString(w, b.String())
	})
}

func (r *Registry) gather() []family {
	r.mu.RLock()
	collectors := append([]collector(nil), r.collectors...)
	r.mu.RUnlock()

	out := make([]family, 0, len(collectors)*familiesPerCollector)
	for _, c := range collectors {
		out = append(out, c()...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}
