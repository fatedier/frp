// Copyright 2025 The frp Authors
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

package vnet

import (
	"context"
	"io"

	"github.com/fatedier/golib/pool"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	offset = 16
)

type TunDevice interface {
	io.ReadWriteCloser
}

func OpenTun(ctx context.Context, addr string) (TunDevice, error) {
	td, err := openTun(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &tunDeviceWrapper{dev: td}, nil
}

type tunDeviceWrapper struct {
	dev tun.Device
}

func (d *tunDeviceWrapper) Read(p []byte) (int, error) {
	buf := pool.GetBuf(len(p) + offset)
	defer pool.PutBuf(buf)

	sz := make([]int, 1)

	n, err := d.dev.Read([][]byte{buf}, sz, offset)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}

	dataSize := sz[0]
	if dataSize > len(p) {
		dataSize = len(p)
	}
	copy(p, buf[offset:offset+dataSize])
	return dataSize, nil
}

func (d *tunDeviceWrapper) Write(p []byte) (int, error) {
	buf := pool.GetBuf(len(p) + offset)
	defer pool.PutBuf(buf)

	copy(buf[offset:], p)
	return d.dev.Write([][]byte{buf}, offset)
}

func (d *tunDeviceWrapper) Close() error {
	return d.dev.Close()
}
