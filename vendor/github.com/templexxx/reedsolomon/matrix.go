package reedsolomon

import "errors"

type matrix []byte

func genEncMatrixCauchy(d, p int) matrix {
	t := d + p
	m := make([]byte, t*d)
	for i := 0; i < d; i++ {
		m[i*d+i] = byte(1)
	}

	d2 := d * d
	for i := d; i < t; i++ {
		for j := 0; j < d; j++ {
			d := i ^ j
			a := inverseTbl[d]
			m[d2] = byte(a)
			d2++
		}
	}
	return m
}

func gfExp(b byte, n int) byte {
	if n == 0 {
		return 1
	}
	if b == 0 {
		return 0
	}
	a := logTbl[b]
	ret := int(a) * n
	for ret >= 255 {
		ret -= 255
	}
	return byte(expTbl[ret])
}

func genVandMatrix(vm []byte, t, d int) {
	for i := 0; i < t; i++ {
		for j := 0; j < d; j++ {
			vm[i*d+j] = gfExp(byte(i), j)
		}
	}
}

func (m matrix) mul(right matrix, rows, cols int, r []byte) {
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			var v byte
			for k := 0; k < cols; k++ {
				v ^= gfMul(m[i*cols+k], right[k*cols+j])
			}
			r[i*cols+j] = v
		}
	}
}

func genEncMatrixVand(d, p int) (matrix, error) {
	t := d + p
	buf := make([]byte, (2*t+4*d)*d)
	vm := buf[:t*d]
	genVandMatrix(vm, t, d)
	top := buf[t*d : (t+d)*d]
	copy(top, vm[:d*d])
	raw := buf[(t+d)*d : (t+3*d)*d]
	im := buf[(t+3*d)*d : (t+4*d)*d]
	err := matrix(top).invert(raw, d, im)
	if err != nil {
		return nil, err
	}
	r := buf[(t+4*d)*d : (2*t+4*d)*d]
	matrix(vm).mul(im, t, d, r)
	return matrix(r), nil
}

// [I|m'] -> [m']
func (m matrix) subMatrix(n int, r []byte) {
	for i := 0; i < n; i++ {
		off := i * n
		copy(r[off:off+n], m[2*off+n:2*(off+n)])
	}
}

func (m matrix) invert(raw matrix, n int, im []byte) error {
	// [m] -> [m|I]
	for i := 0; i < n; i++ {
		t := i * n
		copy(raw[2*t:2*t+n], m[t:t+n])
		raw[2*t+i+n] = byte(1)
	}
	err := gauss(raw, n)
	if err != nil {
		return err
	}
	raw.subMatrix(n, im)
	return nil
}

func (m matrix) swap(i, j, n int) {
	for k := 0; k < n; k++ {
		m[i*n+k], m[j*n+k] = m[j*n+k], m[i*n+k]
	}
}

func gfMul(a, b byte) byte {
	return mulTbl[a][b]
}

var errSingular = errors.New("rs.invert: matrix is singular")

// [m|I] -> [I|m']
func gauss(m matrix, n int) error {
	n2 := 2 * n
	for i := 0; i < n; i++ {
		if m[i*n2+i] == 0 {
			for j := i + 1; j < n; j++ {
				if m[j*n2+i] != 0 {
					m.swap(i, j, n2)
					break
				}
			}
		}
		if m[i*n2+i] == 0 {
			return errSingular
		}
		if m[i*n2+i] != 1 {
			d := m[i*n2+i]
			scale := inverseTbl[d]
			for c := 0; c < n2; c++ {
				m[i*n2+c] = gfMul(m[i*n2+c], scale)
			}
		}
		for j := i + 1; j < n; j++ {
			if m[j*n2+i] != 0 {
				scale := m[j*n2+i]
				for c := 0; c < n2; c++ {
					m[j*n2+c] ^= gfMul(scale, m[i*n2+c])
				}
			}
		}
	}
	for k := 0; k < n; k++ {
		for j := 0; j < k; j++ {
			if m[j*n2+k] != 0 {
				scale := m[j*n2+k]
				for c := 0; c < n2; c++ {
					m[j*n2+c] ^= gfMul(scale, m[k*n2+c])
				}
			}
		}
	}
	return nil
}
