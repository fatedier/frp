/**
 * Reed-Solomon Coding over 8-bit values.
 *
 * Copyright 2015, Klaus Post
 * Copyright 2015, Backblaze, Inc.
 */

package reedsolomon

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
)

// StreamEncoder is an interface to encode Reed-Salomon parity sets for your data.
// It provides a fully streaming interface, and processes data in blocks of up to 4MB.
//
// For small shard sizes, 10MB and below, it is recommended to use the in-memory interface,
// since the streaming interface has a start up overhead.
//
// For all operations, no readers and writers should not assume any order/size of
// individual reads/writes.
//
// For usage examples, see "stream-encoder.go" and "streamdecoder.go" in the examples
// folder.
type StreamEncoder interface {
	// Encode parity shards for a set of data shards.
	//
	// Input is 'shards' containing readers for data shards followed by parity shards
	// io.Writer.
	//
	// The number of shards must match the number given to NewStream().
	//
	// Each reader must supply the same number of bytes.
	//
	// The parity shards will be written to the writer.
	// The number of bytes written will match the input size.
	//
	// If a data stream returns an error, a StreamReadError type error
	// will be returned. If a parity writer returns an error, a
	// StreamWriteError will be returned.
	Encode(data []io.Reader, parity []io.Writer) error

	// Verify returns true if the parity shards contain correct data.
	//
	// The number of shards must match the number total data+parity shards
	// given to NewStream().
	//
	// Each reader must supply the same number of bytes.
	// If a shard stream returns an error, a StreamReadError type error
	// will be returned.
	Verify(shards []io.Reader) (bool, error)

	// Reconstruct will recreate the missing shards if possible.
	//
	// Given a list of valid shards (to read) and invalid shards (to write)
	//
	// You indicate that a shard is missing by setting it to nil in the 'valid'
	// slice and at the same time setting a non-nil writer in "fill".
	// An index cannot contain both non-nil 'valid' and 'fill' entry.
	// If both are provided 'ErrReconstructMismatch' is returned.
	//
	// If there are too few shards to reconstruct the missing
	// ones, ErrTooFewShards will be returned.
	//
	// The reconstructed shard set is complete, but integrity is not verified.
	// Use the Verify function to check if data set is ok.
	Reconstruct(valid []io.Reader, fill []io.Writer) error

	// Split a an input stream into the number of shards given to the encoder.
	//
	// The data will be split into equally sized shards.
	// If the data size isn't dividable by the number of shards,
	// the last shard will contain extra zeros.
	//
	// You must supply the total size of your input.
	// 'ErrShortData' will be returned if it is unable to retrieve the
	// number of bytes indicated.
	Split(data io.Reader, dst []io.Writer, size int64) (err error)

	// Join the shards and write the data segment to dst.
	//
	// Only the data shards are considered.
	//
	// You must supply the exact output size you want.
	// If there are to few shards given, ErrTooFewShards will be returned.
	// If the total data size is less than outSize, ErrShortData will be returned.
	Join(dst io.Writer, shards []io.Reader, outSize int64) error
}

// StreamReadError is returned when a read error is encountered
// that relates to a supplied stream.
// This will allow you to find out which reader has failed.
type StreamReadError struct {
	Err    error // The error
	Stream int   // The stream number on which the error occurred
}

// Error returns the error as a string
func (s StreamReadError) Error() string {
	return fmt.Sprintf("error reading stream %d: %s", s.Stream, s.Err)
}

// String returns the error as a string
func (s StreamReadError) String() string {
	return s.Error()
}

// StreamWriteError is returned when a write error is encountered
// that relates to a supplied stream. This will allow you to
// find out which reader has failed.
type StreamWriteError struct {
	Err    error // The error
	Stream int   // The stream number on which the error occurred
}

// Error returns the error as a string
func (s StreamWriteError) Error() string {
	return fmt.Sprintf("error writing stream %d: %s", s.Stream, s.Err)
}

// String returns the error as a string
func (s StreamWriteError) String() string {
	return s.Error()
}

// rsStream contains a matrix for a specific
// distribution of datashards and parity shards.
// Construct if using NewStream()
type rsStream struct {
	r  *reedSolomon
	bs int // Block size
	// Shard reader
	readShards func(dst [][]byte, in []io.Reader) error
	// Shard writer
	writeShards func(out []io.Writer, in [][]byte) error
	creads      bool
	cwrites     bool
}

// NewStream creates a new encoder and initializes it to
// the number of data shards and parity shards that
// you want to use. You can reuse this encoder.
// Note that the maximum number of data shards is 256.
func NewStream(dataShards, parityShards int, o ...Option) (StreamEncoder, error) {
	enc, err := New(dataShards, parityShards, o...)
	if err != nil {
		return nil, err
	}
	rs := enc.(*reedSolomon)
	r := rsStream{r: rs, bs: 4 << 20}
	r.readShards = readShards
	r.writeShards = writeShards
	return &r, err
}

// NewStreamC creates a new encoder and initializes it to
// the number of data shards and parity shards given.
//
// This functions as 'NewStream', but allows you to enable CONCURRENT reads and writes.
func NewStreamC(dataShards, parityShards int, conReads, conWrites bool, o ...Option) (StreamEncoder, error) {
	enc, err := New(dataShards, parityShards, o...)
	if err != nil {
		return nil, err
	}
	rs := enc.(*reedSolomon)
	r := rsStream{r: rs, bs: 4 << 20}
	r.readShards = readShards
	r.writeShards = writeShards
	if conReads {
		r.readShards = cReadShards
	}
	if conWrites {
		r.writeShards = cWriteShards
	}
	return &r, err
}

func createSlice(n, length int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		out[i] = make([]byte, length)
	}
	return out
}

// Encodes parity shards for a set of data shards.
//
// Input is 'shards' containing readers for data shards followed by parity shards
// io.Writer.
//
// The number of shards must match the number given to NewStream().
//
// Each reader must supply the same number of bytes.
//
// The parity shards will be written to the writer.
// The number of bytes written will match the input size.
//
// If a data stream returns an error, a StreamReadError type error
// will be returned. If a parity writer returns an error, a
// StreamWriteError will be returned.
func (r rsStream) Encode(data []io.Reader, parity []io.Writer) error {
	if len(data) != r.r.DataShards {
		return ErrTooFewShards
	}

	if len(parity) != r.r.ParityShards {
		return ErrTooFewShards
	}

	all := createSlice(r.r.Shards, r.bs)
	in := all[:r.r.DataShards]
	out := all[r.r.DataShards:]
	read := 0

	for {
		err := r.readShards(in, data)
		switch err {
		case nil:
		case io.EOF:
			if read == 0 {
				return ErrShardNoData
			}
			return nil
		default:
			return err
		}
		out = trimShards(out, shardSize(in))
		read += shardSize(in)
		err = r.r.Encode(all)
		if err != nil {
			return err
		}
		err = r.writeShards(parity, out)
		if err != nil {
			return err
		}
	}
}

// Trim the shards so they are all the same size
func trimShards(in [][]byte, size int) [][]byte {
	for i := range in {
		if in[i] != nil {
			in[i] = in[i][0:size]
		}
		if len(in[i]) < size {
			in[i] = nil
		}
	}
	return in
}

func readShards(dst [][]byte, in []io.Reader) error {
	if len(in) != len(dst) {
		panic("internal error: in and dst size do not match")
	}
	size := -1
	for i := range in {
		if in[i] == nil {
			dst[i] = nil
			continue
		}
		n, err := io.ReadFull(in[i], dst[i])
		// The error is EOF only if no bytes were read.
		// If an EOF happens after reading some but not all the bytes,
		// ReadFull returns ErrUnexpectedEOF.
		switch err {
		case io.ErrUnexpectedEOF, io.EOF:
			if size < 0 {
				size = n
			} else if n != size {
				// Shard sizes must match.
				return ErrShardSize
			}
			dst[i] = dst[i][0:n]
		case nil:
			continue
		default:
			return StreamReadError{Err: err, Stream: i}
		}
	}
	if size == 0 {
		return io.EOF
	}
	return nil
}

func writeShards(out []io.Writer, in [][]byte) error {
	if len(out) != len(in) {
		panic("internal error: in and out size do not match")
	}
	for i := range in {
		if out[i] == nil {
			continue
		}
		n, err := out[i].Write(in[i])
		if err != nil {
			return StreamWriteError{Err: err, Stream: i}
		}
		//
		if n != len(in[i]) {
			return StreamWriteError{Err: io.ErrShortWrite, Stream: i}
		}
	}
	return nil
}

type readResult struct {
	n    int
	size int
	err  error
}

// cReadShards reads shards concurrently
func cReadShards(dst [][]byte, in []io.Reader) error {
	if len(in) != len(dst) {
		panic("internal error: in and dst size do not match")
	}
	var wg sync.WaitGroup
	wg.Add(len(in))
	res := make(chan readResult, len(in))
	for i := range in {
		if in[i] == nil {
			dst[i] = nil
			wg.Done()
			continue
		}
		go func(i int) {
			defer wg.Done()
			n, err := io.ReadFull(in[i], dst[i])
			// The error is EOF only if no bytes were read.
			// If an EOF happens after reading some but not all the bytes,
			// ReadFull returns ErrUnexpectedEOF.
			res <- readResult{size: n, err: err, n: i}

		}(i)
	}
	wg.Wait()
	close(res)
	size := -1
	for r := range res {
		switch r.err {
		case io.ErrUnexpectedEOF, io.EOF:
			if size < 0 {
				size = r.size
			} else if r.size != size {
				// Shard sizes must match.
				return ErrShardSize
			}
			dst[r.n] = dst[r.n][0:r.size]
		case nil:
		default:
			return StreamReadError{Err: r.err, Stream: r.n}
		}
	}
	if size == 0 {
		return io.EOF
	}
	return nil
}

// cWriteShards writes shards concurrently
func cWriteShards(out []io.Writer, in [][]byte) error {
	if len(out) != len(in) {
		panic("internal error: in and out size do not match")
	}
	var errs = make(chan error, len(out))
	var wg sync.WaitGroup
	wg.Add(len(out))
	for i := range in {
		go func(i int) {
			defer wg.Done()
			if out[i] == nil {
				errs <- nil
				return
			}
			n, err := out[i].Write(in[i])
			if err != nil {
				errs <- StreamWriteError{Err: err, Stream: i}
				return
			}
			if n != len(in[i]) {
				errs <- StreamWriteError{Err: io.ErrShortWrite, Stream: i}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

// Verify returns true if the parity shards contain correct data.
//
// The number of shards must match the number total data+parity shards
// given to NewStream().
//
// Each reader must supply the same number of bytes.
// If a shard stream returns an error, a StreamReadError type error
// will be returned.
func (r rsStream) Verify(shards []io.Reader) (bool, error) {
	if len(shards) != r.r.Shards {
		return false, ErrTooFewShards
	}

	read := 0
	all := createSlice(r.r.Shards, r.bs)
	for {
		err := r.readShards(all, shards)
		if err == io.EOF {
			if read == 0 {
				return false, ErrShardNoData
			}
			return true, nil
		}
		if err != nil {
			return false, err
		}
		read += shardSize(all)
		ok, err := r.r.Verify(all)
		if !ok || err != nil {
			return ok, err
		}
	}
}

// ErrReconstructMismatch is returned by the StreamEncoder, if you supply
// "valid" and "fill" streams on the same index.
// Therefore it is impossible to see if you consider the shard valid
// or would like to have it reconstructed.
var ErrReconstructMismatch = errors.New("valid shards and fill shards are mutually exclusive")

// Reconstruct will recreate the missing shards if possible.
//
// Given a list of valid shards (to read) and invalid shards (to write)
//
// You indicate that a shard is missing by setting it to nil in the 'valid'
// slice and at the same time setting a non-nil writer in "fill".
// An index cannot contain both non-nil 'valid' and 'fill' entry.
//
// If there are too few shards to reconstruct the missing
// ones, ErrTooFewShards will be returned.
//
// The reconstructed shard set is complete when explicitly asked for all missing shards.
// However its integrity is not automatically verified.
// Use the Verify function to check in case the data set is complete.
func (r rsStream) Reconstruct(valid []io.Reader, fill []io.Writer) error {
	if len(valid) != r.r.Shards {
		return ErrTooFewShards
	}
	if len(fill) != r.r.Shards {
		return ErrTooFewShards
	}

	all := createSlice(r.r.Shards, r.bs)
	reconDataOnly := true
	for i := range valid {
		if valid[i] != nil && fill[i] != nil {
			return ErrReconstructMismatch
		}
		if i >= r.r.DataShards && fill[i] != nil {
			reconDataOnly = false
		}
	}

	read := 0
	for {
		err := r.readShards(all, valid)
		if err == io.EOF {
			if read == 0 {
				return ErrShardNoData
			}
			return nil
		}
		if err != nil {
			return err
		}
		read += shardSize(all)
		all = trimShards(all, shardSize(all))

		if reconDataOnly {
			err = r.r.ReconstructData(all) // just reconstruct missing data shards
		} else {
			err = r.r.Reconstruct(all) //  reconstruct all missing shards
		}
		if err != nil {
			return err
		}
		err = r.writeShards(fill, all)
		if err != nil {
			return err
		}
	}
}

// Join the shards and write the data segment to dst.
//
// Only the data shards are considered.
//
// You must supply the exact output size you want.
// If there are to few shards given, ErrTooFewShards will be returned.
// If the total data size is less than outSize, ErrShortData will be returned.
func (r rsStream) Join(dst io.Writer, shards []io.Reader, outSize int64) error {
	// Do we have enough shards?
	if len(shards) < r.r.DataShards {
		return ErrTooFewShards
	}

	// Trim off parity shards if any
	shards = shards[:r.r.DataShards]
	for i := range shards {
		if shards[i] == nil {
			return StreamReadError{Err: ErrShardNoData, Stream: i}
		}
	}
	// Join all shards
	src := io.MultiReader(shards...)

	// Copy data to dst
	n, err := io.CopyN(dst, src, outSize)
	if err == io.EOF {
		return ErrShortData
	}
	if err != nil {
		return err
	}
	if n != outSize {
		return ErrShortData
	}
	return nil
}

// Split a an input stream into the number of shards given to the encoder.
//
// The data will be split into equally sized shards.
// If the data size isn't dividable by the number of shards,
// the last shard will contain extra zeros.
//
// You must supply the total size of your input.
// 'ErrShortData' will be returned if it is unable to retrieve the
// number of bytes indicated.
func (r rsStream) Split(data io.Reader, dst []io.Writer, size int64) error {
	if size == 0 {
		return ErrShortData
	}
	if len(dst) != r.r.DataShards {
		return ErrInvShardNum
	}

	for i := range dst {
		if dst[i] == nil {
			return StreamWriteError{Err: ErrShardNoData, Stream: i}
		}
	}

	// Calculate number of bytes per shard.
	perShard := (size + int64(r.r.DataShards) - 1) / int64(r.r.DataShards)

	// Pad data to r.Shards*perShard.
	padding := make([]byte, (int64(r.r.Shards)*perShard)-size)
	data = io.MultiReader(data, bytes.NewBuffer(padding))

	// Split into equal-length shards and copy.
	for i := range dst {
		n, err := io.CopyN(dst[i], data, perShard)
		if err != io.EOF && err != nil {
			return err
		}
		if n != perShard {
			return ErrShortData
		}
	}

	return nil
}
