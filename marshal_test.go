package hll

import (
	"bytes"
	crand "crypto/rand"
	"encoding/gob"
	"encoding/json"
	mrand "math/rand"
	"testing"

	"github.com/bmizerany/assert"
)

func TestMarshalRoundTrip(t *testing.T) {
	const p, pPrime = 14, 25

	testCases := []struct {
		p, pPrime uint
	}{
		{5, 10},
		{10, 25},
		{15, 25},
	}

	for _, testCase := range testCases {
		h := NewHll(testCase.p, testCase.pPrime)
		for i := uint64(0); i <= 1e5; i++ {
			if i%5000 == 0 {
				// Every N elements, do a round-trip marshal and unmarshal and make sure cardinality is
				// preserved.
				jBuf, err := json.Marshal(h)
				assert.Equalf(t, nil, err, "%v", err)

				rt := &Hll{}
				err = json.Unmarshal(jBuf, rt)
				assert.Equalf(t, nil, err, "%v", err)

				assert.Equal(t, rt.Cardinality(), h.Cardinality())
			}

			h.Add(randUint64(t))
		}
		assert.T(t, !h.isSparse) // Ensure we stored enough to use the dense representation.
	}
}

// Make sure that after roundtripping, an Hll is still usable and behaves identically.
func TestUsageAfterMarshalRoundTrip(t *testing.T) {
	h := NewHll(10, 20)

	h.Add(randUint64(t))
	h.Add(randUint64(t))
	h.Add(randUint64(t))

	jBuf, err := json.Marshal(h)
	assert.Equalf(t, nil, err, "%v", err)

	rt := &Hll{}
	err = json.Unmarshal(jBuf, rt)
	assert.Equalf(t, nil, err, "%v", err)

	for i := uint64(100); i < 1000; i++ {
		r := randUint64(t)
		rt.Add(r)
		h.Add(r)

		assert.Equalf(t, rt.isSparse, h.isSparse, "%v", i)
		// fmt.Printf("2 calling rt.Cardinality(), rt.isSparse=%v\n", rt.isSparse)
		rtCard := rt.Cardinality()
		// fmt.Printf("3\n")
		hCard := h.Cardinality()
		assert.Equal(t, rtCard, hCard, i)
	}

	assert.T(t, !h.isSparse)
}

// The JSON-marshaled form of an Hll should include either a sparseList or a dense/normal register
// array, and not both. This checks whether we got the JSON library omitempty usage right.
func TestMarshalOmit(t *testing.T) {
	h := NewHll(10, 25)

	check := func() {
		jBuf, err := json.Marshal(h)
		assert.Equal(t, nil, err)
		m := map[string]interface{}{}
		err = json.Unmarshal(jBuf, &m)
		assert.Equal(t, nil, err)

		_, hasDense := m["M"]
		_, hasSparse := m["s"]

		assert.T(t, hasDense || hasSparse)
		assert.T(t, !(hasDense && hasSparse))
	}

	for h.isSparse {
		h.Add(randUint64(t))
		check() // This checks the sparse case.
	}

	check() // This checks the dense case.
}

func TestMarshalPbRoundtrip(t *testing.T) {
	const p, pPrime = 14, 25

	testCases := []struct {
		p, pPrime uint
	}{
		{5, 10},
		{10, 25},
		{15, 25},
	}

	for _, testCase := range testCases {
		h := NewHll(testCase.p, testCase.pPrime)
		for i := uint64(0); i <= 1e5; i++ {
			if i%5000 == 0 {
				// Every N elements, do a round-trip marshal and unmarshal and make sure cardinality is
				// preserved.
				pbBuf, err := h.MarshalPb()
				assert.Equalf(t, nil, err, "%v", err)

				rt := &Hll{}
				err = rt.UnmarshalPb(pbBuf)
				assert.Equalf(t, nil, err, "%v", err)

				assert.Equal(t, rt.Cardinality(), h.Cardinality())
			}

			h.Add(randUint64(t))
		}

		assert.T(t, !h.isSparse) // Ensure we stored enough to use the dense representation.
	}
}

func TestMarshalGobRoundTrip(t *testing.T) {
	const p, pPrime = 14, 25

	testCases := []struct {
		p, pPrime uint
	}{
		{5, 10},
		{10, 25},
		{15, 25},
	}

	for _, testCase := range testCases {
		h := NewHll(testCase.p, testCase.pPrime)
		for i := uint64(0); i <= 1e5; i++ {
			if i%5000 == 0 {
				// Every N elements, do a round-trip marshal and unmarshal and make sure cardinality is
				// preserved.
				var val bytes.Buffer
				enc := gob.NewEncoder(&val)
				err := enc.Encode(h)
				assert.Equal(t, err, nil)

				// decode
				dec := gob.NewDecoder(&val)
				rt := &Hll{}
				err = dec.Decode(rt)
				assert.Equal(t, err, nil)

				assert.Equal(t, rt.Cardinality(), h.Cardinality())
			}

			h.Add(randUint64(t))
		}

		assert.T(t, !h.isSparse) // Ensure we stored enough to use the dense representation.
	}
}

func TestCompression(t *testing.T) {
	const numTests = 1000

	for i := 0; i < numTests; i++ {
		numToRead := mrand.Intn(100)
		buf := make([]byte, numToRead)
		n, err := crand.Read(buf)
		assert.Equal(t, nil, err)
		assert.Equal(t, n, numToRead)

		compressed, err := snappyB64(buf)
		assert.Equal(t, nil, err)
		roundTripped, err := unsnappyB64(compressed)
		assert.Equal(t, nil, err)

		assert.Equal(t, buf, roundTripped)
	}
}
