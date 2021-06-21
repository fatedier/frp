package rpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

func WriteBytes(w io.Writer, buf []byte) (int, error) {
	out := bytes.NewBuffer(nil)
	binary.Write(out, binary.BigEndian, int64(len(buf)))
	out.Write(buf)
	return w.Write(out.Bytes())
}

func ReadBytes(r io.Reader) ([]byte, error) {
	var length int64
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	buffer := make([]byte, length)
	n, err := io.ReadFull(r, buffer)
	if err != nil {
		return nil, err
	}
	if int64(n) != length {
		return nil, errors.New("invalid length")
	}
	return buffer, nil
}
