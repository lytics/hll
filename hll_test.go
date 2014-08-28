package hll

import (
	"math"
	"testing"

	"github.com/bmizerany/assert"
)

// adding values in the dense case
// determine if the maximum rho value for a determined index is correctly written to M.
func TestAddNormal(t *testing.T) {
	h := NewHll(14, 20)

	value := uint64(0xAABBCCDD00112210)
	value2 := uint64(0xAABBCCDD00112211)

	register := value >> (64 - h.p)
	register2 := value2 >> (64 - h.p)
	assert.Equal(t, register2, register)
	assert.T(t, rho(value) > rho(value2))

	h.switchToNormal()
	h.addNormal(value)
	h.addNormal(value2)
	assert.Equal(t, h.bigM.Get(uint64(register)), rho(value))
}

// Check to make sure that the temp set gets merged when it's supposed to
// and if it changes to the dense represtation if it passes the sparse threshold
func TestAddSparse(t *testing.T) {
	h := NewHll(14, 20)

	assert.Equal(t, h.isSparse, true)
	// the maximum size of the sparseList is 6145: (2^18) * (6/4) / 64
	rands := randUint64s(t, 6145)

	for idx, randVal := range rands {
		h.Add(randVal)

		// tempSet should be reset after adding (2^p * (6 / 4) / 64 elements
		if uint64(idx*64)%h.mergeSizeBits == 1 {
			assert.Equal(t, len(h.tempSet), 1)
		}

		if h.isSparse == false {
			assert.T(t, h.sparseList == nil)
			assert.Equal(t, h.tempSet, []uint64{})
			break
		}
	}
}

// Tests cardinality accuracy with varying number of distinct uint64 inputs
func TestCardinality(t *testing.T) {
	// number of random values to estimate cardinalities for
	counts := []int{1000, 5000, 20000, 50000, 100000, 250000, 1000000, 10000000}

	for _, count := range counts {
		// Create new Hll struct with p = 14 & p' = 25
		h := NewHll(14, 25)
		// Random uint64 values to test.
		rands := randUint64s(t, count)

		// startTime := time.Now()
		for _, randomU64 := range rands {
			h.Add(randomU64)
		}
		card := h.Cardinality()

		calculatedError := math.Abs(float64(card)-float64(count)) / float64(count)
		assert.T(t, calculatedError < 0.15)
		// endTime := time.Since(startTime)
		//	fmt.Printf("\nActual Cardinality: %d\n Estimated Cardinality: %d\nError: %v\nTime Elapsed: %v\n\n", count, card, calculatedError, endTime)
	}
}

// Test the weighted mean estimate for the bias for precision 4.
func TestEstimateBias(t *testing.T) {
	h_four := NewHll(4, 10)

	// according to empirical bias calculations, bias should be below 9.2 and above 8.78
	bias := h_four.estimateBias(12.5)
	assert.T(t, bias > 8.78 && bias < 9.20)

	// if estimate is not in the estimated range, return max bias
	max_bias := h_four.estimateBias(80.00)
	assert.Equal(t, max_bias, -1.7606)
}

func TestCombine(t *testing.T) {
	testCases := []struct {
		p, pPrime      uint
		count1, count2 int
		shouldBeSparse bool
	}{
		{12, 25, 50, 100, true},
		{12, 25, 5000, 10000, false},
		{12, 25, 5, 10000, false},
		{12, 25, 10000, 5, false},
	}

	for i, testCase := range testCases {
		inputs1 := randUint64s(t, testCase.count1)
		inputs2 := randUint64s(t, testCase.count2)

		hll1 := NewHll(testCase.p, testCase.pPrime)
		for _, x := range inputs1 {
			hll1.Add(x)
		}

		hll2 := NewHll(testCase.p, testCase.pPrime)
		for _, x := range inputs2 {
			hll2.Add(x)
		}

		hll1.Combine(hll2)
		if testCase.shouldBeSparse != hll1.isSparse {
			t.Errorf("Testcase %d: expected isSparse %v but was %v", i, testCase.shouldBeSparse,
				hll1.isSparse)
		}

		expectedCard := float64(len(inputs1) + len(inputs2))
		estimatedCard := float64(hll1.Cardinality())
		wrongness := math.Abs(estimatedCard-expectedCard)/estimatedCard - 1
		if wrongness >= 0.05 {
			t.Errorf("Testcase %d: cardinality wrongness %v was too high", i, wrongness)
		}
	}
}
