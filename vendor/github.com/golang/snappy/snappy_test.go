// Copyright 2011 The Snappy-Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snappy

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	download = flag.Bool("download", false, "If true, download any missing files before running benchmarks")
	testdata = flag.String("testdata", "testdata", "Directory containing the test data")
)

func TestMaxEncodedLenOfMaxUncompressedChunkLen(t *testing.T) {
	got := maxEncodedLenOfMaxUncompressedChunkLen
	want := MaxEncodedLen(maxUncompressedChunkLen)
	if got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
}

func roundtrip(b, ebuf, dbuf []byte) error {
	d, err := Decode(dbuf, Encode(ebuf, b))
	if err != nil {
		return fmt.Errorf("decoding error: %v", err)
	}
	if !bytes.Equal(b, d) {
		return fmt.Errorf("roundtrip mismatch:\n\twant %v\n\tgot  %v", b, d)
	}
	return nil
}

func TestEmpty(t *testing.T) {
	if err := roundtrip(nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSmallCopy(t *testing.T) {
	for _, ebuf := range [][]byte{nil, make([]byte, 20), make([]byte, 64)} {
		for _, dbuf := range [][]byte{nil, make([]byte, 20), make([]byte, 64)} {
			for i := 0; i < 32; i++ {
				s := "aaaa" + strings.Repeat("b", i) + "aaaabbbb"
				if err := roundtrip([]byte(s), ebuf, dbuf); err != nil {
					t.Errorf("len(ebuf)=%d, len(dbuf)=%d, i=%d: %v", len(ebuf), len(dbuf), i, err)
				}
			}
		}
	}
}

func TestSmallRand(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for n := 1; n < 20000; n += 23 {
		b := make([]byte, n)
		for i := range b {
			b[i] = uint8(rng.Uint32())
		}
		if err := roundtrip(b, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSmallRegular(t *testing.T) {
	for n := 1; n < 20000; n += 23 {
		b := make([]byte, n)
		for i := range b {
			b[i] = uint8(i%10 + 'a')
		}
		if err := roundtrip(b, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInvalidVarint(t *testing.T) {
	data := []byte("\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\x00")
	if _, err := DecodedLen(data); err != ErrCorrupt {
		t.Errorf("DecodedLen: got %v, want ErrCorrupt", err)
	}
	if _, err := Decode(nil, data); err != ErrCorrupt {
		t.Errorf("Decode: got %v, want ErrCorrupt", err)
	}

	// The encoded varint overflows 32 bits
	data = []byte("\xff\xff\xff\xff\xff\x00")

	if _, err := DecodedLen(data); err != ErrCorrupt {
		t.Errorf("DecodedLen: got %v, want ErrCorrupt", err)
	}
	if _, err := Decode(nil, data); err != ErrCorrupt {
		t.Errorf("Decode: got %v, want ErrCorrupt", err)
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		desc    string
		input   string
		want    string
		wantErr error
	}{{
		`decodedLen=0x100000000 is too long`,
		"\x80\x80\x80\x80\x10" + "\x00\x41",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 0-byte length; length=3; valid input`,
		"\x03" + "\x08\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=1; tagLiteral, 1-byte length; not enough length bytes`,
		"\x01" + "\xf0",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 1-byte length; length=3; valid input`,
		"\x03" + "\xf0\x02\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=1; tagLiteral, 2-byte length; not enough length bytes`,
		"\x01" + "\xf4\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 2-byte length; length=3; valid input`,
		"\x03" + "\xf4\x02\x00\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=1; tagLiteral, 3-byte length; not enough length bytes`,
		"\x01" + "\xf8\x00\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 3-byte length; length=3; valid input`,
		"\x03" + "\xf8\x02\x00\x00\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=1; tagLiteral, 4-byte length; not enough length bytes`,
		"\x01" + "\xfc\x00\x00\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=1; tagLiteral, 4-byte length; length=3; not enough dst bytes`,
		"\x01" + "\xfc\x02\x00\x00\x00\xff\xff\xff",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=4; tagLiteral, 4-byte length; length=3; not enough src bytes`,
		"\x04" + "\xfc\x02\x00\x00\x00\xff",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=3; tagLiteral, 4-byte length; length=3; valid input`,
		"\x03" + "\xfc\x02\x00\x00\x00\xff\xff\xff",
		"\xff\xff\xff",
		nil,
	}, {
		`decodedLen=4; tagCopy1, 1 extra length|offset byte; not enough extra bytes`,
		"\x04" + "\x01",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=4; tagCopy2, 2 extra length|offset bytes; not enough extra bytes`,
		"\x04" + "\x02\x00",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=4; tagCopy4; unsupported COPY_4 tag`,
		"\x04" + "\x03\x00\x00\x00\x00",
		"",
		errUnsupportedCopy4Tag,
	}, {
		`decodedLen=4; tagLiteral (4 bytes "abcd"); valid input`,
		"\x04" + "\x0cabcd",
		"abcd",
		nil,
	}, {
		`decodedLen=8; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=4; valid input`,
		"\x08" + "\x0cabcd" + "\x01\x04",
		"abcdabcd",
		nil,
	}, {
		`decodedLen=9; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=4; inconsistent dLen`,
		"\x09" + "\x0cabcd" + "\x01\x04",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=8; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=5; offset too large`,
		"\x08" + "\x0cabcd" + "\x01\x05",
		"",
		ErrCorrupt,
	}, {
		`decodedLen=7; tagLiteral (4 bytes "abcd"); tagCopy1; length=4 offset=4; length too large`,
		"\x07" + "\x0cabcd" + "\x01\x04",
		"",
		ErrCorrupt,
	}}

	for _, tc := range testCases {
		g, gotErr := Decode(nil, []byte(tc.input))
		if got := string(g); got != tc.want || gotErr != tc.wantErr {
			t.Errorf("%s:\ngot  %q, %v\nwant %q, %v", tc.desc, got, gotErr, tc.want, tc.wantErr)
		}
	}
}

func cmp(a, b []byte) error {
	if len(a) != len(b) {
		return fmt.Errorf("got %d bytes, want %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("byte #%d: got 0x%02x, want 0x%02x", i, a[i], b[i])
		}
	}
	return nil
}

func TestFramingFormat(t *testing.T) {
	// src is comprised of alternating 1e5-sized sequences of random
	// (incompressible) bytes and repeated (compressible) bytes. 1e5 was chosen
	// because it is larger than maxUncompressedChunkLen (64k).
	src := make([]byte, 1e6)
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			for j := 0; j < 1e5; j++ {
				src[1e5*i+j] = uint8(rng.Intn(256))
			}
		} else {
			for j := 0; j < 1e5; j++ {
				src[1e5*i+j] = uint8(i)
			}
		}
	}

	buf := new(bytes.Buffer)
	if _, err := NewWriter(buf).Write(src); err != nil {
		t.Fatalf("Write: encoding: %v", err)
	}
	dst, err := ioutil.ReadAll(NewReader(buf))
	if err != nil {
		t.Fatalf("ReadAll: decoding: %v", err)
	}
	if err := cmp(dst, src); err != nil {
		t.Fatal(err)
	}
}

func TestWriterGoldenOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	w := NewBufferedWriter(buf)
	defer w.Close()
	w.Write([]byte("abcd")) // Not compressible.
	w.Flush()
	w.Write(bytes.Repeat([]byte{'A'}, 100)) // Compressible.
	w.Flush()
	got := buf.String()
	want := strings.Join([]string{
		magicChunk,
		"\x01\x08\x00\x00", // Uncompressed chunk, 8 bytes long (including 4 byte checksum).
		"\x68\x10\xe6\xb6", // Checksum.
		"\x61\x62\x63\x64", // Uncompressed payload: "abcd".
		"\x00\x0d\x00\x00", // Compressed chunk, 13 bytes long (including 4 byte checksum).
		"\x37\xcb\xbc\x9d", // Checksum.
		"\x64",             // Compressed payload: Uncompressed length (varint encoded): 100.
		"\x00\x41",         // Compressed payload: tagLiteral, length=1, "A".
		"\xfe\x01\x00",     // Compressed payload: tagCopy2,   length=64, offset=1.
		"\x8a\x01\x00",     // Compressed payload: tagCopy2,   length=35, offset=1.
	}, "")
	if got != want {
		t.Fatalf("\ngot:  % x\nwant: % x", got, want)
	}
}

func TestNewBufferedWriter(t *testing.T) {
	// Test all 32 possible sub-sequences of these 5 input slices.
	//
	// Their lengths sum to 400,000, which is over 6 times the Writer ibuf
	// capacity: 6 * maxUncompressedChunkLen is 393,216.
	inputs := [][]byte{
		bytes.Repeat([]byte{'a'}, 40000),
		bytes.Repeat([]byte{'b'}, 150000),
		bytes.Repeat([]byte{'c'}, 60000),
		bytes.Repeat([]byte{'d'}, 120000),
		bytes.Repeat([]byte{'e'}, 30000),
	}
loop:
	for i := 0; i < 1<<uint(len(inputs)); i++ {
		var want []byte
		buf := new(bytes.Buffer)
		w := NewBufferedWriter(buf)
		for j, input := range inputs {
			if i&(1<<uint(j)) == 0 {
				continue
			}
			if _, err := w.Write(input); err != nil {
				t.Errorf("i=%#02x: j=%d: Write: %v", i, j, err)
				continue loop
			}
			want = append(want, input...)
		}
		if err := w.Close(); err != nil {
			t.Errorf("i=%#02x: Close: %v", i, err)
			continue
		}
		got, err := ioutil.ReadAll(NewReader(buf))
		if err != nil {
			t.Errorf("i=%#02x: ReadAll: %v", i, err)
			continue
		}
		if err := cmp(got, want); err != nil {
			t.Errorf("i=%#02x: %v", i, err)
			continue
		}
	}
}

func TestFlush(t *testing.T) {
	buf := new(bytes.Buffer)
	w := NewBufferedWriter(buf)
	defer w.Close()
	if _, err := w.Write(bytes.Repeat([]byte{'x'}, 20)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n := buf.Len(); n != 0 {
		t.Fatalf("before Flush: %d bytes were written to the underlying io.Writer, want 0", n)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if n := buf.Len(); n == 0 {
		t.Fatalf("after Flush: %d bytes were written to the underlying io.Writer, want non-0", n)
	}
}

func TestReaderReset(t *testing.T) {
	gold := bytes.Repeat([]byte("All that is gold does not glitter,\n"), 10000)
	buf := new(bytes.Buffer)
	if _, err := NewWriter(buf).Write(gold); err != nil {
		t.Fatalf("Write: %v", err)
	}
	encoded, invalid, partial := buf.String(), "invalid", "partial"
	r := NewReader(nil)
	for i, s := range []string{encoded, invalid, partial, encoded, partial, invalid, encoded, encoded} {
		if s == partial {
			r.Reset(strings.NewReader(encoded))
			if _, err := r.Read(make([]byte, 101)); err != nil {
				t.Errorf("#%d: %v", i, err)
				continue
			}
			continue
		}
		r.Reset(strings.NewReader(s))
		got, err := ioutil.ReadAll(r)
		switch s {
		case encoded:
			if err != nil {
				t.Errorf("#%d: %v", i, err)
				continue
			}
			if err := cmp(got, gold); err != nil {
				t.Errorf("#%d: %v", i, err)
				continue
			}
		case invalid:
			if err == nil {
				t.Errorf("#%d: got nil error, want non-nil", i)
				continue
			}
		}
	}
}

func TestWriterReset(t *testing.T) {
	gold := bytes.Repeat([]byte("Not all those who wander are lost;\n"), 10000)
	const n = 20
	for _, buffered := range []bool{false, true} {
		var w *Writer
		if buffered {
			w = NewBufferedWriter(nil)
			defer w.Close()
		} else {
			w = NewWriter(nil)
		}

		var gots, wants [][]byte
		failed := false
		for i := 0; i <= n; i++ {
			buf := new(bytes.Buffer)
			w.Reset(buf)
			want := gold[:len(gold)*i/n]
			if _, err := w.Write(want); err != nil {
				t.Errorf("#%d: Write: %v", i, err)
				failed = true
				continue
			}
			if buffered {
				if err := w.Flush(); err != nil {
					t.Errorf("#%d: Flush: %v", i, err)
					failed = true
					continue
				}
			}
			got, err := ioutil.ReadAll(NewReader(buf))
			if err != nil {
				t.Errorf("#%d: ReadAll: %v", i, err)
				failed = true
				continue
			}
			gots = append(gots, got)
			wants = append(wants, want)
		}
		if failed {
			continue
		}
		for i := range gots {
			if err := cmp(gots[i], wants[i]); err != nil {
				t.Errorf("#%d: %v", i, err)
			}
		}
	}
}

func TestWriterResetWithoutFlush(t *testing.T) {
	buf0 := new(bytes.Buffer)
	buf1 := new(bytes.Buffer)
	w := NewBufferedWriter(buf0)
	if _, err := w.Write([]byte("xxx")); err != nil {
		t.Fatalf("Write #0: %v", err)
	}
	// Note that we don't Flush the Writer before calling Reset.
	w.Reset(buf1)
	if _, err := w.Write([]byte("yyy")); err != nil {
		t.Fatalf("Write #1: %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	got, err := ioutil.ReadAll(NewReader(buf1))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if err := cmp(got, []byte("yyy")); err != nil {
		t.Fatal(err)
	}
}

type writeCounter int

func (c *writeCounter) Write(p []byte) (int, error) {
	*c++
	return len(p), nil
}

// TestNumUnderlyingWrites tests that each Writer flush only makes one or two
// Write calls on its underlying io.Writer, depending on whether or not the
// flushed buffer was compressible.
func TestNumUnderlyingWrites(t *testing.T) {
	testCases := []struct {
		input []byte
		want  int
	}{
		{bytes.Repeat([]byte{'x'}, 100), 1},
		{bytes.Repeat([]byte{'y'}, 100), 1},
		{[]byte("ABCDEFGHIJKLMNOPQRST"), 2},
	}

	var c writeCounter
	w := NewBufferedWriter(&c)
	defer w.Close()
	for i, tc := range testCases {
		c = 0
		if _, err := w.Write(tc.input); err != nil {
			t.Errorf("#%d: Write: %v", i, err)
			continue
		}
		if err := w.Flush(); err != nil {
			t.Errorf("#%d: Flush: %v", i, err)
			continue
		}
		if int(c) != tc.want {
			t.Errorf("#%d: got %d underlying writes, want %d", i, c, tc.want)
			continue
		}
	}
}

func benchDecode(b *testing.B, src []byte) {
	encoded := Encode(nil, src)
	// Bandwidth is in amount of uncompressed data.
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(src, encoded)
	}
}

func benchEncode(b *testing.B, src []byte) {
	// Bandwidth is in amount of uncompressed data.
	b.SetBytes(int64(len(src)))
	dst := make([]byte, MaxEncodedLen(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Encode(dst, src)
	}
}

func readFile(b testing.TB, filename string) []byte {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		b.Skipf("skipping benchmark: %v", err)
	}
	if len(src) == 0 {
		b.Fatalf("%s has zero length", filename)
	}
	return src
}

// expand returns a slice of length n containing repeated copies of src.
func expand(src []byte, n int) []byte {
	dst := make([]byte, n)
	for x := dst; len(x) > 0; {
		i := copy(x, src)
		x = x[i:]
	}
	return dst
}

func benchWords(b *testing.B, n int, decode bool) {
	// Note: the file is OS-language dependent so the resulting values are not
	// directly comparable for non-US-English OS installations.
	data := expand(readFile(b, "/usr/share/dict/words"), n)
	if decode {
		benchDecode(b, data)
	} else {
		benchEncode(b, data)
	}
}

func BenchmarkWordsDecode1e1(b *testing.B) { benchWords(b, 1e1, true) }
func BenchmarkWordsDecode1e2(b *testing.B) { benchWords(b, 1e2, true) }
func BenchmarkWordsDecode1e3(b *testing.B) { benchWords(b, 1e3, true) }
func BenchmarkWordsDecode1e4(b *testing.B) { benchWords(b, 1e4, true) }
func BenchmarkWordsDecode1e5(b *testing.B) { benchWords(b, 1e5, true) }
func BenchmarkWordsDecode1e6(b *testing.B) { benchWords(b, 1e6, true) }
func BenchmarkWordsEncode1e1(b *testing.B) { benchWords(b, 1e1, false) }
func BenchmarkWordsEncode1e2(b *testing.B) { benchWords(b, 1e2, false) }
func BenchmarkWordsEncode1e3(b *testing.B) { benchWords(b, 1e3, false) }
func BenchmarkWordsEncode1e4(b *testing.B) { benchWords(b, 1e4, false) }
func BenchmarkWordsEncode1e5(b *testing.B) { benchWords(b, 1e5, false) }
func BenchmarkWordsEncode1e6(b *testing.B) { benchWords(b, 1e6, false) }

func BenchmarkRandomEncode(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = uint8(rng.Intn(256))
	}
	benchEncode(b, data)
}

// testFiles' values are copied directly from
// https://raw.githubusercontent.com/google/snappy/master/snappy_unittest.cc
// The label field is unused in snappy-go.
//
// If this list changes (due to the upstream C++ list changing), remember to
// update the .gitignore file in this repository.
var testFiles = []struct {
	label     string
	filename  string
	sizeLimit int
}{
	{"html", "html", 0},
	{"urls", "urls.10K", 0},
	{"jpg", "fireworks.jpeg", 0},
	{"jpg_200", "fireworks.jpeg", 200},
	{"pdf", "paper-100k.pdf", 0},
	{"html4", "html_x_4", 0},
	{"txt1", "alice29.txt", 0},
	{"txt2", "asyoulik.txt", 0},
	{"txt3", "lcet10.txt", 0},
	{"txt4", "plrabn12.txt", 0},
	{"pb", "geo.protodata", 0},
	{"gaviota", "kppkn.gtb", 0},
}

// The test data files are present at this canonical URL.
const baseURL = "https://raw.githubusercontent.com/google/snappy/master/testdata/"

func downloadTestdata(b *testing.B, basename string) (errRet error) {
	filename := filepath.Join(*testdata, basename)
	if stat, err := os.Stat(filename); err == nil && stat.Size() != 0 {
		return nil
	}

	if !*download {
		b.Skipf("test data not found; skipping benchmark without the -download flag")
	}
	// Download the official snappy C++ implementation reference test data
	// files for benchmarking.
	if err := os.Mkdir(*testdata, 0777); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create testdata: %s", err)
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s: %s", filename, err)
	}
	defer f.Close()
	defer func() {
		if errRet != nil {
			os.Remove(filename)
		}
	}()
	url := baseURL + basename
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %s", url, err)
	}
	defer resp.Body.Close()
	if s := resp.StatusCode; s != http.StatusOK {
		return fmt.Errorf("downloading %s: HTTP status code %d (%s)", url, s, http.StatusText(s))
	}
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download %s to %s: %s", url, filename, err)
	}
	return nil
}

func benchFile(b *testing.B, n int, decode bool) {
	if err := downloadTestdata(b, testFiles[n].filename); err != nil {
		b.Fatalf("failed to download testdata: %s", err)
	}
	data := readFile(b, filepath.Join(*testdata, testFiles[n].filename))
	if n := testFiles[n].sizeLimit; 0 < n && n < len(data) {
		data = data[:n]
	}
	if decode {
		benchDecode(b, data)
	} else {
		benchEncode(b, data)
	}
}

// Naming convention is kept similar to what snappy's C++ implementation uses.
func Benchmark_UFlat0(b *testing.B)  { benchFile(b, 0, true) }
func Benchmark_UFlat1(b *testing.B)  { benchFile(b, 1, true) }
func Benchmark_UFlat2(b *testing.B)  { benchFile(b, 2, true) }
func Benchmark_UFlat3(b *testing.B)  { benchFile(b, 3, true) }
func Benchmark_UFlat4(b *testing.B)  { benchFile(b, 4, true) }
func Benchmark_UFlat5(b *testing.B)  { benchFile(b, 5, true) }
func Benchmark_UFlat6(b *testing.B)  { benchFile(b, 6, true) }
func Benchmark_UFlat7(b *testing.B)  { benchFile(b, 7, true) }
func Benchmark_UFlat8(b *testing.B)  { benchFile(b, 8, true) }
func Benchmark_UFlat9(b *testing.B)  { benchFile(b, 9, true) }
func Benchmark_UFlat10(b *testing.B) { benchFile(b, 10, true) }
func Benchmark_UFlat11(b *testing.B) { benchFile(b, 11, true) }
func Benchmark_ZFlat0(b *testing.B)  { benchFile(b, 0, false) }
func Benchmark_ZFlat1(b *testing.B)  { benchFile(b, 1, false) }
func Benchmark_ZFlat2(b *testing.B)  { benchFile(b, 2, false) }
func Benchmark_ZFlat3(b *testing.B)  { benchFile(b, 3, false) }
func Benchmark_ZFlat4(b *testing.B)  { benchFile(b, 4, false) }
func Benchmark_ZFlat5(b *testing.B)  { benchFile(b, 5, false) }
func Benchmark_ZFlat6(b *testing.B)  { benchFile(b, 6, false) }
func Benchmark_ZFlat7(b *testing.B)  { benchFile(b, 7, false) }
func Benchmark_ZFlat8(b *testing.B)  { benchFile(b, 8, false) }
func Benchmark_ZFlat9(b *testing.B)  { benchFile(b, 9, false) }
func Benchmark_ZFlat10(b *testing.B) { benchFile(b, 10, false) }
func Benchmark_ZFlat11(b *testing.B) { benchFile(b, 11, false) }
