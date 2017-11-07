// +build !amd64

package reedsolomon

func newRS(d, p int, em matrix) (enc Encoder) {
	g := em[d*d:]
	return &encBase{data: d, parity: p, encode: em, gen: g}
}
