package rpc

import (
	"bufio"
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
	// To compatible with UDP connection, use bufio reader here to avoid lost conent.
	rd := bufio.NewReader(r)

	var length int64
	if err := binary.Read(rd, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	buffer := make([]byte, length)
	n, err := io.ReadFull(rd, buffer)
	if err != nil {
		return nil, err
	}
	if int64(n) != length {
		return nil, errors.New("invalid length")
	}
	return buffer, nil
}
