package hll

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"github.com/golang/snappy"
)

type sparse struct {
	buf                  []byte
	lastVal, numElements uint64
}

func newSparse(estimatedCap uint64) *sparse {
	return &sparse{
		buf:         make([]byte, 0, estimatedCap),
		lastVal:     0,
		numElements: 0,
	}
}

func (s *sparse) Copy() *sparse {
	if s == nil {
		return nil
	}
	buf := make([]byte, len(s.buf))
	copy(buf, s.buf)
	return &sparse{
		buf:         buf,
		lastVal:     s.lastVal,
		numElements: s.numElements,
	}
}

func (s *sparse) Add(x uint64) {
	delta := x - s.lastVal

	// This slice is not strictly necessary, but it saves a lot of complexity. For now, simplicity
	// trumps performance in this case.
	deltaBuf := make([]byte, binary.MaxVarintLen64)

	n := binary.PutUvarint(deltaBuf, delta)
	s.buf = append(s.buf, deltaBuf[0:n]...)
	s.lastVal = x
	s.numElements++
}

func (s *sparse) SizeInBits() uint64 {
	return uint64(len(s.buf) * 8)
}

func (s *sparse) SizeInBytes() uint64 {
	return uint64(len(s.buf))
}

// Returns a function that can be called repeatedly to yield values from the list.
func (s *sparse) GetIterator() u64It {
	// idx := 0
	rdr := bytes.NewBuffer(s.buf)
	var lastDecoded uint64 = 0
	return func() (uint64, bool) {
		delta, err := binary.ReadUvarint(rdr)
		if err != nil {
			return 0, false
		}
		returnVal := lastDecoded + delta
		lastDecoded = returnVal
		return returnVal, true
	}
}

func (s *sparse) GetNumElements() uint64 {
	return s.numElements
}

type jsonableSparse struct {
	B    []byte
	L, N uint64
}

func (s *sparse) MarshalJSON() ([]byte, error) {
	compressed, err := snappyB64(s.buf)
	if err != nil {
		return nil, err
	}
	j := &jsonableSparse{compressed, s.lastVal, s.numElements}
	return json.Marshal(j)
}

func (s *sparse) UnmarshalJSON(buf []byte) error {
	j := jsonableSparse{}
	if err := json.Unmarshal(buf, &j); err != nil {
		return err
	}

	uncompressed, err := unsnappyB64(j.B)
	if err != nil {
		return err
	}

	s.buf, s.lastVal, s.numElements = uncompressed, j.L, j.N
	return nil
}

// Compress the input using snapp and encode the result using URL-safe base64.
func snappyB64(in []byte) ([]byte, error) {
	compressed := snappy.Encode(nil, in)
	outBuf := make([]byte, base64.URLEncoding.EncodedLen(len(compressed)))
	base64.URLEncoding.Encode(outBuf, compressed)
	return outBuf, nil
}

// The inverse of snappyB64.
func unsnappyB64(in []byte) ([]byte, error) {
	unBase64ed := make([]byte, base64.URLEncoding.DecodedLen(len(in)))
	n, err := base64.URLEncoding.Decode(unBase64ed, in)
	if err != nil {
		return nil, err
	}

	uncompressed, err := snappy.Decode(nil, unBase64ed[:n])
	if err != nil {
		return nil, err
	}

	// The snappy library returns nil when the output length is zero. Fix it now.
	// I filed this bug upstream: https://code.google.com/p/snappy-go/issues/detail?id=6
	if uncompressed == nil {
		uncompressed = []byte{}
	}
	return uncompressed, nil
}
