/**
 * Unit tests for Galois
 *
 * Copyright 2015, Klaus Post
 * Copyright 2015, Backblaze, Inc.
 */

package reedsolomon

import (
	"bytes"
	"testing"
)

func TestAssociativity(t *testing.T) {
	for i := 0; i < 256; i++ {
		a := byte(i)
		for j := 0; j < 256; j++ {
			b := byte(j)
			for k := 0; k < 256; k++ {
				c := byte(k)
				x := galAdd(a, galAdd(b, c))
				y := galAdd(galAdd(a, b), c)
				if x != y {
					t.Fatal("add does not match:", x, "!=", y)
				}
				x = galMultiply(a, galMultiply(b, c))
				y = galMultiply(galMultiply(a, b), c)
				if x != y {
					t.Fatal("multiply does not match:", x, "!=", y)
				}
			}
		}
	}
}

func TestIdentity(t *testing.T) {
	for i := 0; i < 256; i++ {
		a := byte(i)
		b := galAdd(a, 0)
		if a != b {
			t.Fatal("Add zero should yield same result", a, "!=", b)
		}
		b = galMultiply(a, 1)
		if a != b {
			t.Fatal("Mul by one should yield same result", a, "!=", b)
		}
	}
}

func TestInverse(t *testing.T) {
	for i := 0; i < 256; i++ {
		a := byte(i)
		b := galSub(0, a)
		c := galAdd(a, b)
		if c != 0 {
			t.Fatal("inverse sub/add", c, "!=", 0)
		}
		if a != 0 {
			b = galDivide(1, a)
			c = galMultiply(a, b)
			if c != 1 {
				t.Fatal("inverse div/mul", c, "!=", 1)
			}
		}
	}
}

func TestCommutativity(t *testing.T) {
	for i := 0; i < 256; i++ {
		a := byte(i)
		for j := 0; j < 256; j++ {
			b := byte(j)
			x := galAdd(a, b)
			y := galAdd(b, a)
			if x != y {
				t.Fatal(x, "!= ", y)
			}
			x = galMultiply(a, b)
			y = galMultiply(b, a)
			if x != y {
				t.Fatal(x, "!= ", y)
			}
		}
	}
}

func TestDistributivity(t *testing.T) {
	for i := 0; i < 256; i++ {
		a := byte(i)
		for j := 0; j < 256; j++ {
			b := byte(j)
			for k := 0; k < 256; k++ {
				c := byte(k)
				x := galMultiply(a, galAdd(b, c))
				y := galAdd(galMultiply(a, b), galMultiply(a, c))
				if x != y {
					t.Fatal(x, "!= ", y)
				}
			}
		}
	}
}

func TestExp(t *testing.T) {
	for i := 0; i < 256; i++ {
		a := byte(i)
		power := byte(1)
		for j := 0; j < 256; j++ {
			x := galExp(a, j)
			if x != power {
				t.Fatal(x, "!=", power)
			}
			power = galMultiply(power, a)
		}
	}
}

func TestGalois(t *testing.T) {
	// These values were copied output of the Python code.
	if galMultiply(3, 4) != 12 {
		t.Fatal("galMultiply(3, 4) != 12")
	}
	if galMultiply(7, 7) != 21 {
		t.Fatal("galMultiply(7, 7) != 21")
	}
	if galMultiply(23, 45) != 41 {
		t.Fatal("galMultiply(23, 45) != 41")
	}

	// Test slices (>16 entries to test assembler)
	in := []byte{0, 1, 2, 3, 4, 5, 6, 10, 50, 100, 150, 174, 201, 255, 99, 32, 67, 85}
	out := make([]byte, len(in))
	galMulSlice(25, in, out, false, false)
	expect := []byte{0x0, 0x19, 0x32, 0x2b, 0x64, 0x7d, 0x56, 0xfa, 0xb8, 0x6d, 0xc7, 0x85, 0xc3, 0x1f, 0x22, 0x7, 0x25, 0xfe}
	if 0 != bytes.Compare(out, expect) {
		t.Errorf("got %#v, expected %#v", out, expect)
	}

	galMulSlice(177, in, out, false, false)
	expect = []byte{0x0, 0xb1, 0x7f, 0xce, 0xfe, 0x4f, 0x81, 0x9e, 0x3, 0x6, 0xe8, 0x75, 0xbd, 0x40, 0x36, 0xa3, 0x95, 0xcb}
	if 0 != bytes.Compare(out, expect) {
		t.Errorf("got %#v, expected %#v", out, expect)
	}

	if galExp(2, 2) != 4 {
		t.Fatal("galExp(2, 2) != 4")
	}
	if galExp(5, 20) != 235 {
		t.Fatal("galExp(5, 20) != 235")
	}
	if galExp(13, 7) != 43 {
		t.Fatal("galExp(13, 7) != 43")
	}
}
