// Copyright 2017 fatedier, fatedier@gmail.com
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

package crypto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrypto(t *testing.T) {
	assert := assert.New(t)

	text := "1234567890abcdefghigklmnopqrstuvwxyzeeeeeeeeeeeeeeeeeeeeeewwwwwwwwwwwwwwwwwwwwwwwwwwzzzzzzzzzzzzzzzzzzzzzzzzdddddddddddddddddddddddddddddddddddddrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrllllllllllllllllllllllllllllllllllqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeewwwwwwwwwwwwwwwwwwwwww"
	key := "123456"

	buffer := bytes.NewBuffer(nil)
	encWriter, err := NewWriter(buffer, []byte(key))
	assert.NoError(err)
	decReader := NewReader(buffer, []byte(key))

	encWriter.Write([]byte(text))

	c := bytes.NewBuffer(nil)
	io.Copy(c, decReader)
	assert.Equal(text, string(c.Bytes()))
}
