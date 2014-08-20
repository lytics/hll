package hll

import (
	"bytes"
	"encoding/binary"
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
