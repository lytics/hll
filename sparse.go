package hll

import (
	"code.google.com/p/goprotobuf/proto"
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

func (s *sparse) Add(x uint64) {
	delta := x - s.lastVal
	deltaBuf := proto.EncodeVarint(delta)
	s.buf = append(s.buf, deltaBuf...)
	s.lastVal = x
	s.numElements++
}

func (s *sparse) SizeInBits() uint64 {
	return uint64(len(s.buf) * 8)
}

// Returns a function that can be called repeatedly to yield values from the list.
func (s *sparse) GetIterator() func() (_ uint64, ok bool) {
	idx := 0
	var lastDecoded uint64 = 0
	return func() (uint64, bool) {
		delta, n := proto.DecodeVarint(s.buf[idx:])
		if n == 0 {
			return 0, false
		}
		idx += n
		returnVal := lastDecoded + delta
		lastDecoded = returnVal
		return returnVal, true
	}
}

func (s *sparse) GetNumElements() uint64 {
	return s.numElements
}
