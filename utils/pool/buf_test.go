// Copyright 2016 fatedier, fatedier@gmail.com
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

package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPutBuf(t *testing.T) {
	buf := make([]byte, 512)
	PutBuf(buf)

	buf = make([]byte, 1025)
	PutBuf(buf)

	buf = make([]byte, 2*1025)
	PutBuf(buf)

	buf = make([]byte, 5*1025)
	PutBuf(buf)
}

func TestGetBuf(t *testing.T) {
	assert := assert.New(t)

	buf := GetBuf(200)
	assert.Len(buf, 200)

	buf = GetBuf(1025)
	assert.Len(buf, 1025)

	buf = GetBuf(2 * 1024)
	assert.Len(buf, 2*1024)

	buf = GetBuf(5 * 2000)
	assert.Len(buf, 5*2000)
}
