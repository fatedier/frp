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

package limit

import (
	"fmt"

	"golang.org/x/time/rate"
)

// NewBandwidthLimiter creates a limiter whose rate preserves the configured
// byte limit while keeping the burst representable as an int on all targets.
func NewBandwidthLimiter(bytes int64) *rate.Limiter {
	if bytes <= 0 {
		return nil
	}

	maxInt := int64(^uint(0) >> 1)
	burst := min(bytes, maxInt)
	return rate.NewLimiter(rate.Limit(float64(bytes)), int(burst))
}

func invalidBurstError(burst int) error {
	return fmt.Errorf("invalid limiter burst: %d", burst)
}
