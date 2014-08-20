package hll

import (
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/bmizerany/assert"
)

func TestSparseIterator(t *testing.T) {
	s := newSparse(5)
	inputs := []uint64{3, 5, 6, 6, 10}
	for _, x := range inputs {
		s.Add(x)
	}
	iter := s.GetIterator()
	for _, elem := range inputs {
		iterOutput, ok := iter()
		assert.T(t, ok)
		assert.Equal(t, uint64(elem), iterOutput)
	}
	_, ok := iter()
	assert.T(t, !ok) // iterator should be exhausted
}

func TestRho(t *testing.T) {
	testCases := []struct {
		input        uint64
		expectResult uint8
	}{
		{1, 1},
		{0, 63},
		{4, 3},
	}

	for i, testCase := range testCases {
		actualResult := rho(testCase.input)
		if testCase.expectResult != actualResult {
			t.Errorf("Case %d actual result was %v", i, actualResult)
		}
	}
}

func Dedupe(input []uint64, p, pPrime uint) []uint64 {
	var output []uint64
	for idx, value := range input {
		if idx > 0 && getIndex(value, p, pPrime) == getIndex(input[idx-1], p, pPrime) {
			continue
		}
		output = append(output, value)
	}
	return output
}

func TestMerge(t *testing.T) {
	const p = 12
	const pPrime = 25

	encodeHashes := func(vals []uint64) []uint64 {
		s := make([]uint64, len(vals))
		for i, v := range vals {
			encoded := encodeHash(v, p, pPrime)
			s[i] = encoded
		}
		sortHashcodesByIndex([]uint64(s), p, pPrime)
		// fmt.Printf("encodeHashes: returning %x\n", s)
		return s
	}

	encodeCompressed := func(vals []uint64) *sparse {
		cs := newSparse(0)
		var encoded_hashes []uint64
		for _, v := range vals {
			encoded := encodeHash(v, p, pPrime)
			encoded_hashes = append(encoded_hashes, encoded)
		}
		sortHashcodesByIndex(encoded_hashes, p, pPrime)
		deduped := Dedupe(encoded_hashes, p, pPrime)
		for _, hash := range deduped {
			cs.Add(hash)
		}
		return cs
	}

	compressedInput := randUint64s(t, 200)
	sortHashcodesByIndex(compressedInput, p, pPrime)
	firstInput := encodeCompressed(compressedInput)
	secondInput := encodeHashes(randUint64s(t, 100))

	merged := merge(firstInput, secondInput, p, pPrime)

	var lastIndex uint

	mergedIter := merged.GetIterator()
	value, valid := mergedIter()
	for valid {
		index, _ := decodeHash(value, p, pPrime)
		assert.T(t, index > lastIndex, index, lastIndex)
		lastIndex = index
		value, valid = mergedIter()
	}
}

func randUint64s(t *testing.T, count int) []uint64 {
	buf := make([]byte, 8)
	output := make([]uint64, count)
	for i := 0; i < count; i++ {
		n, err := rand.Read(buf)
		assert.T(t, err == nil && n == 8, err, n)
		// fmt.Printf("random buf: %x\n", buf)
		output[i] = binary.LittleEndian.Uint64(buf)
	}
	return output
}
