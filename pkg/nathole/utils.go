// Copyright 2023 The frp Authors
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

package nathole

import (
	"bytes"

	"github.com/fatedier/golib/crypto"

	"github.com/fatedier/frp/pkg/msg"
)

func EncodeMessage(m msg.Message, key []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := msg.WriteMsg(buffer, m); err != nil {
		return nil, err
	}

	buf, err := crypto.Encode(buffer.Bytes(), key)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func DecodeMessageInto(data, key []byte, m msg.Message) error {
	buf, err := crypto.Decode(data, key)
	if err != nil {
		return err
	}

	if err := msg.ReadMsgInto(bytes.NewReader(buf), m); err != nil {
		return err
	}
	return nil
}
