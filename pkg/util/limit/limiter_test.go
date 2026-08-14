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
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestNewBandwidthLimiterClampsBurstToTargetInt(t *testing.T) {
	const bytesPerSecond = int64(1 << 31)

	limiter := NewBandwidthLimiter(bytesPerSecond)
	require.NotNil(t, limiter)

	wantBurst := bytesPerSecond
	maxInt := int64(^uint(0) >> 1)
	if wantBurst > maxInt {
		wantBurst = maxInt
	}
	require.Equal(t, int(wantBurst), limiter.Burst())
	require.Equal(t, rate.Limit(float64(bytesPerSecond)), limiter.Limit())
}

func TestNewBandwidthLimiterDisablesNonPositiveLimit(t *testing.T) {
	require.Nil(t, NewBandwidthLimiter(0))
	require.Nil(t, NewBandwidthLimiter(-1))
}

func TestReaderAndWriterRejectInvalidBurst(t *testing.T) {
	for _, burst := range []int{0, -1} {
		t.Run("reader/"+strconv.Itoa(burst), func(t *testing.T) {
			reader := NewReader(strings.NewReader("payload"), rate.NewLimiter(rate.Limit(1), burst))
			n, err := reader.Read(make([]byte, 1))
			require.Zero(t, n)
			require.EqualError(t, err, "invalid limiter burst: "+strconv.Itoa(burst))
		})

		t.Run("writer/"+strconv.Itoa(burst), func(t *testing.T) {
			var dst bytes.Buffer
			writer := NewWriter(&dst, rate.NewLimiter(rate.Limit(1), burst))
			n, err := writer.Write([]byte("payload"))
			require.Zero(t, n)
			require.EqualError(t, err, "invalid limiter burst: "+strconv.Itoa(burst))
			require.Empty(t, dst.Bytes())
		})
	}
}
