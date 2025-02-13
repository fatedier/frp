package rpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func WriteBytes(w io.Writer, buf []byte) (int, error) {
	out := bytes.NewBuffer(nil)
	if err := binary.Write(out, binary.BigEndian, int64(len(buf))); err != nil {
		return 0, err
	}

	out.Write(buf)
	return w.Write(out.Bytes())
}

func ReadBytes(r io.Reader) ([]byte, error) {
	var length int64
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length < 0 || length > 10*1024*1024 {
		return nil, fmt.Errorf("invalid length")
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
