// Copyright 2015 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package context

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	//Default size==20B same as nginx
	defaultGzipMinLength = 20
	//Content will only be compressed if content length is either unknown or greater than gzipMinLength.
	gzipMinLength = defaultGzipMinLength
	//The compression level used for deflate compression. (0-9).
	gzipCompressLevel int
	//List of HTTP methods to compress. If not set, only GET requests are compressed.
	includedMethods map[string]bool
	getMethodOnly   bool
)

func InitGzip(minLength, compressLevel int, methods []string) {
	if minLength >= 0 {
		gzipMinLength = minLength
	}
	gzipCompressLevel = compressLevel
	if gzipCompressLevel < flate.NoCompression || gzipCompressLevel > flate.BestCompression {
		gzipCompressLevel = flate.BestSpeed
	}
	getMethodOnly = (len(methods) == 0) || (len(methods) == 1 && strings.ToUpper(methods[0]) == "GET")
	includedMethods = make(map[string]bool, len(methods))
	for _, v := range methods {
		includedMethods[strings.ToUpper(v)] = true
	}
}

type resetWriter interface {
	io.Writer
	Reset(w io.Writer)
}

type nopResetWriter struct {
	io.Writer
}

func (n nopResetWriter) Reset(w io.Writer) {
	//do nothing
}

type acceptEncoder struct {
	name                    string
	levelEncode             func(int) resetWriter
	customCompressLevelPool *sync.Pool
	bestCompressionPool     *sync.Pool
}

func (ac acceptEncoder) encode(wr io.Writer, level int) resetWriter {
	if ac.customCompressLevelPool == nil || ac.bestCompressionPool == nil {
		return nopResetWriter{wr}
	}
	var rwr resetWriter
	switch level {
	case flate.BestSpeed:
		rwr = ac.customCompressLevelPool.Get().(resetWriter)
	case flate.BestCompression:
		rwr = ac.bestCompressionPool.Get().(resetWriter)
	default:
		rwr = ac.levelEncode(level)
	}
	rwr.Reset(wr)
	return rwr
}

func (ac acceptEncoder) put(wr resetWriter, level int) {
	if ac.customCompressLevelPool == nil || ac.bestCompressionPool == nil {
		return
	}
	wr.Reset(nil)

	//notice
	//compressionLevel==BestCompression DOES NOT MATTER
	//sync.Pool will not memory leak

	switch level {
	case gzipCompressLevel:
		ac.customCompressLevelPool.Put(wr)
	case flate.BestCompression:
		ac.bestCompressionPool.Put(wr)
	}
}

var (
	noneCompressEncoder = acceptEncoder{"", nil, nil, nil}
	gzipCompressEncoder = acceptEncoder{
		name:                    "gzip",
		levelEncode:             func(level int) resetWriter { wr, _ := gzip.NewWriterLevel(nil, level); return wr },
		customCompressLevelPool: &sync.Pool{New: func() interface{} { wr, _ := gzip.NewWriterLevel(nil, gzipCompressLevel); return wr }},
		bestCompressionPool:     &sync.Pool{New: func() interface{} { wr, _ := gzip.NewWriterLevel(nil, flate.BestCompression); return wr }},
	}

	//according to the sec :http://tools.ietf.org/html/rfc2616#section-3.5 ,the deflate compress in http is zlib indeed
	//deflate
	//The "zlib" format defined in RFC 1950 [31] in combination with
	//the "deflate" compression mechanism described in RFC 1951 [29].
	deflateCompressEncoder = acceptEncoder{
		name:                    "deflate",
		levelEncode:             func(level int) resetWriter { wr, _ := zlib.NewWriterLevel(nil, level); return wr },
		customCompressLevelPool: &sync.Pool{New: func() interface{} { wr, _ := zlib.NewWriterLevel(nil, gzipCompressLevel); return wr }},
		bestCompressionPool:     &sync.Pool{New: func() interface{} { wr, _ := zlib.NewWriterLevel(nil, flate.BestCompression); return wr }},
	}
)

var (
	encoderMap = map[string]acceptEncoder{ // all the other compress methods will ignore
		"gzip":     gzipCompressEncoder,
		"deflate":  deflateCompressEncoder,
		"*":        gzipCompressEncoder, // * means any compress will accept,we prefer gzip
		"identity": noneCompressEncoder, // identity means none-compress
	}
)

// WriteFile reads from file and writes to writer by the specific encoding(gzip/deflate)
func WriteFile(encoding string, writer io.Writer, file *os.File) (bool, string, error) {
	return writeLevel(encoding, writer, file, flate.BestCompression)
}

// WriteBody reads  writes content to writer by the specific encoding(gzip/deflate)
func WriteBody(encoding string, writer io.Writer, content []byte) (bool, string, error) {
	if encoding == "" || len(content) < gzipMinLength {
		_, err := writer.Write(content)
		return false, "", err
	}
	return writeLevel(encoding, writer, bytes.NewReader(content), gzipCompressLevel)
}

// writeLevel reads from reader,writes to writer by specific encoding and compress level
// the compress level is defined by deflate package
func writeLevel(encoding string, writer io.Writer, reader io.Reader, level int) (bool, string, error) {
	var outputWriter resetWriter
	var err error
	var ce = noneCompressEncoder

	if cf, ok := encoderMap[encoding]; ok {
		ce = cf
	}
	encoding = ce.name
	outputWriter = ce.encode(writer, level)
	defer ce.put(outputWriter, level)

	_, err = io.Copy(outputWriter, reader)
	if err != nil {
		return false, "", err
	}

	switch outputWriter.(type) {
	case io.WriteCloser:
		outputWriter.(io.WriteCloser).Close()
	}
	return encoding != "", encoding, nil
}

// ParseEncoding will extract the right encoding for response
// the Accept-Encoding's sec is here:
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.3
func ParseEncoding(r *http.Request) string {
	if r == nil {
		return ""
	}
	if (getMethodOnly && r.Method == "GET") || includedMethods[r.Method] {
		return parseEncoding(r)
	}
	return ""
}

type q struct {
	name  string
	value float64
}

func parseEncoding(r *http.Request) string {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if acceptEncoding == "" {
		return ""
	}
	var lastQ q
	for _, v := range strings.Split(acceptEncoding, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		vs := strings.Split(v, ";")
		var cf acceptEncoder
		var ok bool
		if cf, ok = encoderMap[vs[0]]; !ok {
			continue
		}
		if len(vs) == 1 {
			return cf.name
		}
		if len(vs) == 2 {
			f, _ := strconv.ParseFloat(strings.Replace(vs[1], "q=", "", -1), 64)
			if f == 0 {
				continue
			}
			if f > lastQ.value {
				lastQ = q{cf.name, f}
			}
		}
	}
	return lastQ.name
}
