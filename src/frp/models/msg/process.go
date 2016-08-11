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

package msg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"frp/models/config"
	"frp/models/metric"
	"frp/utils/conn"
	"frp/utils/log"
	"frp/utils/pcrypto"
	"frp/utils/pool"
)

// will block until connection close
func Join(c1 *conn.Conn, c2 *conn.Conn) {
	var wait sync.WaitGroup
	pipe := func(to *conn.Conn, from *conn.Conn) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		var err error
		_, err = io.Copy(to.TcpConn, from.TcpConn)
		if err != nil {
			log.Warn("join connections error, %v", err)
		}
	}

	wait.Add(2)
	go pipe(c1, c2)
	go pipe(c2, c1)
	wait.Wait()
	return
}

// join two connections and do some operations
func JoinMore(c1 *conn.Conn, c2 *conn.Conn, conf config.BaseConf, needRecord bool) {
	var wait sync.WaitGroup
	encryptPipe := func(from *conn.Conn, to *conn.Conn) {
		defer from.Close()
		defer to.Close()
		defer wait.Done()

		// we don't care about errors here
		pipeEncrypt(from, to, conf, needRecord)
	}

	decryptPipe := func(to *conn.Conn, from *conn.Conn) {
		defer from.Close()
		defer to.Close()
		defer wait.Done()

		// we don't care about errors here
		pipeDecrypt(to, from, conf, needRecord)
	}

	wait.Add(2)
	go encryptPipe(c1, c2)
	go decryptPipe(c2, c1)
	wait.Wait()
	if needRecord {
		metric.CloseConnection(conf.Name)
	}
	log.Debug("ProxyName [%s], One tunnel stopped", conf.Name)
	return
}

func pkgMsg(data []byte) []byte {
	llen := uint32(len(data))
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, llen)
	buf.Write(data)
	return buf.Bytes()
}

func unpkgMsg(data []byte) (int, []byte, []byte) {
	if len(data) < 4 {
		return -1, nil, data
	}
	llen := int(binary.BigEndian.Uint32(data[0:4]))
	// no complete
	if len(data) < llen+4 {
		return -1, nil, data
	}

	return 0, data[4 : llen+4], data[llen+4:]
}

// decrypt msg from reader, then write into writer
func pipeDecrypt(r *conn.Conn, w *conn.Conn, conf config.BaseConf, needRecord bool) (err error) {
	laes := new(pcrypto.Pcrypto)
	key := conf.AuthToken
	if conf.PrivilegeMode {
		key = conf.PrivilegeToken
	}
	if err := laes.Init([]byte(key)); err != nil {
		log.Warn("ProxyName [%s], Pcrypto Init error: %v", conf.Name, err)
		return fmt.Errorf("Pcrypto Init error: %v", err)
	}

	// get []byte from buffer pool
	buf := pool.GetBuf(5*1024 + 4)
	defer pool.PutBuf(buf)

	var left, res []byte
	var cnt int = -1

	// record
	var flowBytes int64 = 0
	if needRecord {
		defer func() {
			metric.AddFlowOut(conf.Name, flowBytes)
		}()
	}

	for {
		// there may be more than 1 package in variable
		// and we read more bytes if unpkgMsg returns an error
		var newBuf []byte
		if cnt < 0 {
			n, err := r.Read(buf)
			if err != nil {
				return err
			}
			newBuf = append(left, buf[0:n]...)
		} else {
			newBuf = left
		}
		cnt, res, left = unpkgMsg(newBuf)
		if cnt < 0 {
			// limit one package length, maximum is 1MB
			if len(res) > 1024*1024 {
				log.Warn("ProxyName [%s], package length exceeds the limit")
				return fmt.Errorf("package length error")
			}
			continue
		}

		// aes
		if conf.UseEncryption {
			res, err = laes.Decrypt(res)
			if err != nil {
				log.Warn("ProxyName [%s], decrypt error, %v", conf.Name, err)
				return fmt.Errorf("Decrypt error: %v", err)
			}
		}
		// gzip
		if conf.UseGzip {
			res, err = laes.Decompression(res)
			if err != nil {
				log.Warn("ProxyName [%s], decompression error, %v", conf.Name, err)
				return fmt.Errorf("Decompression error: %v", err)
			}
		}

		_, err = w.WriteBytes(res)
		if err != nil {
			return err
		}

		if needRecord {
			flowBytes += int64(len(res))
			if flowBytes >= 1024*1024 {
				metric.AddFlowOut(conf.Name, flowBytes)
				flowBytes = 0
			}
		}
	}
	return nil
}

// recvive msg from reader, then encrypt msg into writer
func pipeEncrypt(r *conn.Conn, w *conn.Conn, conf config.BaseConf, needRecord bool) (err error) {
	laes := new(pcrypto.Pcrypto)
	key := conf.AuthToken
	if conf.PrivilegeMode {
		key = conf.PrivilegeToken
	}
	if err := laes.Init([]byte(key)); err != nil {
		log.Warn("ProxyName [%s], Pcrypto Init error: %v", conf.Name, err)
		return fmt.Errorf("Pcrypto Init error: %v", err)
	}

	// record
	var flowBytes int64 = 0
	if needRecord {
		defer func() {
			metric.AddFlowIn(conf.Name, flowBytes)
		}()
	}

	// get []byte from buffer pool
	buf := pool.GetBuf(5*1024 + 4)
	defer pool.PutBuf(buf)

	for {
		n, err := r.Read(buf)
		if err != nil {
			return err
		}
		if needRecord {
			flowBytes += int64(n)
			if flowBytes >= 1024*1024 {
				metric.AddFlowIn(conf.Name, flowBytes)
				flowBytes = 0
			}
		}

		res := buf[0:n]
		// gzip
		if conf.UseGzip {
			res, err = laes.Compression(res)
			if err != nil {
				log.Warn("ProxyName [%s], compression error: %v", conf.Name, err)
				return fmt.Errorf("Compression error: %v", err)
			}
		}
		// aes
		if conf.UseEncryption {
			res, err = laes.Encrypt(res)
			if err != nil {
				log.Warn("ProxyName [%s], encrypt error: %v", conf.Name, err)
				return fmt.Errorf("Encrypt error: %v", err)
			}
		}

		res = pkgMsg(res)
		_, err = w.WriteBytes(res)
		if err != nil {
			return err
		}
	}

	return nil
}
