package reedsolomon

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

const (
	kb         = 1024
	mb         = 1024 * 1024
	testNumIn  = 10
	testNumOut = 4
)

const verifySize = 256 + 32 + 16 + 15

func TestVerifyEncBase(t *testing.T) {
	d := 5
	p := 5
	vects := [][]byte{
		{0, 1},
		{4, 5},
		{2, 3},
		{6, 7},
		{8, 9},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
	}
	em, err := genEncMatrixVand(d, p)
	if err != nil {
		t.Fatal(err)
	}
	g := em[d*d:]
	e := &encBase{data: d, parity: p, gen: g}
	err = e.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}
	if vects[5][0] != 12 || vects[5][1] != 13 {
		t.Fatal("vect 5 mismatch")
	}
	if vects[6][0] != 10 || vects[6][1] != 11 {
		t.Fatal("vect 6 mismatch")
	}
	if vects[7][0] != 14 || vects[7][1] != 15 {
		t.Fatal("vect 7 mismatch")
	}
	if vects[8][0] != 90 || vects[8][1] != 91 {
		t.Fatal("vect 8 mismatch")
	}
	if vects[9][0] != 94 || vects[9][1] != 95 {
		t.Fatal("shard 9 mismatch")
	}
}

func fillRandom(v []byte) {
	for i := 0; i < len(v); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(v) && j < 7; j++ {
			v[i+j] = byte(val)
			val >>= 8
		}
	}
}

func verifyEnc(t *testing.T, d, p int) {
	for i := 1; i <= verifySize; i++ {
		vects1 := make([][]byte, d+p)
		vects2 := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
			copy(vects2[j], vects1[j])
		}
		e, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = e.Encode(vects1)
		if err != nil {
			t.Fatal(err)
		}
		em, err := genEncMatrixVand(d, p)
		if err != nil {
			t.Fatal(err)
		}
		g := em[d*d:]
		e2 := &encBase{data: d, parity: p, gen: g}
		err = e2.Encode(vects2)
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				t.Fatalf("no match enc with encBase; vect: %d; size: %d", k, i)
			}
		}
	}
}

func TestVerifyEnc(t *testing.T) {
	verifyEnc(t, testNumIn, testNumOut)
}

func verifyReconst(t *testing.T, d, p int, lost []int) {
	for i := 1; i <= verifySize; i++ {
		vects1 := make([][]byte, d+p)
		vects2 := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
		}
		e, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = e.Encode(vects1)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < d+p; j++ {
			copy(vects2[j], vects1[j])
		}
		for _, i := range lost {
			vects2[i] = nil
		}
		err = e.Reconstruct(vects2)
		if err != nil {
			t.Fatal(err)
		}
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				t.Fatalf("no match reconst; vect: %d; size: %d", k, i)
			}
		}
	}
}

func TestVerifyReconst(t *testing.T) {
	lost := []int{0, 11, 3, 4}
	verifyReconst(t, testNumIn, testNumOut, lost)
}

func benchEnc(b *testing.B, d, p, size int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	e, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = e.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = e.Encode(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchEncRun(f func(*testing.B, int, int, int), d, p int, size []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range size {
			b.Run(fmt.Sprintf("%d+%d_%d", d, p, s), func(b *testing.B) {
				f(b, d, p, s)
			})
		}
	}
}

func BenchmarkEnc(b *testing.B) {
	s1 := []int{1350}
	b.Run("", benchEncRun(benchEnc, 10, 3, s1))
	s2 := []int{1400, 4 * kb, 64 * kb, mb, 16 * mb}
	b.Run("", benchEncRun(benchEnc, testNumIn, testNumOut, s2))
}

func benchReconst(b *testing.B, d, p, size int, lost []int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	e, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = e.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	for _, i := range lost {
		vects[i] = nil
	}
	err = e.Reconstruct(vects)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, i := range lost {
			vects[i] = nil
		}
		err = e.Reconstruct(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchReconstRun(f func(*testing.B, int, int, int, []int), d, p int, size, lost []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range size {
			b.Run(fmt.Sprintf("%d+%d_%d", d, p, s), func(b *testing.B) {
				f(b, d, p, s, lost)
			})
		}
	}
}

// Reconstruct p vects
func BenchmarkReconst(b *testing.B) {
	l1 := []int{2, 4, 5}
	s1 := []int{1350}
	b.Run("", benchReconstRun(benchReconst, 10, 3, s1, l1))
	l2 := []int{2, 4, 7, 9}
	s2 := []int{1400, 4 * kb, 64 * kb, mb, 16 * mb}
	b.Run("", benchReconstRun(benchReconst, testNumIn, testNumOut, s2, l2))
}

func benchReconstPos(b *testing.B, d, p, size int, has, dLost, pLost []int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	e, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = e.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	err = e.ReconstWithPos(vects, has, dLost, pLost)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = e.ReconstWithPos(vects, has, dLost, pLost)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchReconstPosRun(f func(*testing.B, int, int, int, []int, []int, []int), d, p int, size,
	has, dLost, pLost []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range size {
			b.Run(fmt.Sprintf("%dx%d_%d", d, p, s), func(b *testing.B) {
				f(b, d, p, s, has, dLost, pLost)
			})
		}
	}
}

// Reconstruct p vects with position
func BenchmarkReconstWithPos(b *testing.B) {
	h1 := []int{0, 1, 3, 6, 7, 8, 9, 10, 11, 12}
	d1 := []int{2, 4, 5}
	p1 := []int{}
	s1 := []int{1350}
	b.Run("", benchReconstPosRun(benchReconstPos, 10, 3, s1, h1, d1, p1))
	h2 := []int{0, 1, 3, 5, 6, 8, 10, 11, 12, 13}
	d2 := []int{2, 4, 7, 9}
	p2 := []int{}
	s2 := []int{1400, 4 * kb, 64 * kb, mb, 16 * mb}
	b.Run("", benchReconstPosRun(benchReconstPos, testNumIn, testNumOut, s2, h2, d2, p2))
}
