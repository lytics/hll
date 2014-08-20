package hll

import (
	"math"
	"sort"
)

const (
	alpha_16 = 0.673
	alpha_32 = 0.697
	alpha_64 = 0.709
)

// M is used for the dense case, and registers the rho values for each hashed index
// sparseList and tempSet are used in the sparse case during aggregation.
// (dense and sparse specifications are available in their respective go files)
// alpha is the constant used in the hyperloglog calculation.
// p is the number of precision bits to use for the dense case and p must be within [4..18]
// p_prime specifies the number of precision bits to use for the sparse case, where p' <= 64.
// Google recommends that p' be set to either 20 or 25
// isSparse is a boolean flag that determines when to switch over to the dense case.
// mergeSizeBits and sparseThresholdBits are used when adding values in the sparse case.
type Hll struct {
	M                   normal
	sparseList          *sparse
	tempSet             []uint64
	alpha               float64
	isSparse            bool
	p, pPrime           uint
	m, mPrime           uint64
	mergeSizeBits       uint64
	sparseThresholdBits uint64
}

// Initialize a new hyper-log-log struct.
func NewHll(p, pPrime uint) *Hll {
	if p < 4 || p > 18 {
		panic("p must be in the range [4,18]")
	}

	h := &Hll{}
	h.p = p
	h.pPrime = pPrime
	h.m = 1 << h.p
	h.mPrime = 1 << pPrime
	h.isSparse = true

	switch h.m {
	case 16:
		h.alpha = alpha_16
	case 32:
		h.alpha = alpha_32
	case 64:
		h.alpha = alpha_64
	default:
		h.alpha = 0.7213 / (1.0 + 1.079/float64(h.m))
	}

	h.M = newNormal(h.m)
	h.sparseList = newSparse(0)
	h.tempSet = []uint64{}

	// The sparse threshold (the threshold for when to convert to the normal case) is set to m.6 bits.
	h.sparseThresholdBits = h.m * 6

	// When the temp set reaches 25% of the maximum size for sparse register storage, merge the
	// temp set with the sparse list.
	h.mergeSizeBits = h.sparseThresholdBits / 4

	return h
}

// Aggregation step. The inputs should be hashes, which should be roughly uniformly
// distributed (any good hash function will do).
func (h *Hll) Add(x uint64) {
	if h.isSparse {
		h.addSparse(x)
	} else {
		h.addNormal(x)
	}
}

func (h *Hll) addSparse(x uint64) {
	k := encodeHash(x, h.p, h.pPrime)
	h.tempSet = append(h.tempSet, k)

	tempSetBits := uint64(len(h.tempSet)) * 64
	if tempSetBits > h.mergeSizeBits {
		// temp set sorting is done in merge
		h.sparseList = merge(h.sparseList, h.tempSet, h.p, h.pPrime)
		h.tempSet = []uint64{}
		if h.sparseList.SizeInBits() > h.sparseThresholdBits {
			h.isSparse = false
			h.M = toNormal(h.sparseList, h.p, h.pPrime)
		}
	}
}

func (h *Hll) addNormal(x uint64) {
	offset := (64 - h.p)
	idx := x >> offset
	r := rho(x)
	if r > h.M.Get(idx) {
		h.M.Set(idx, r)
	}
}

// Returns Cardinality Estimate according to current state (sparse or normal).
func (h *Hll) Cardinality() uint64 {
	if h.isSparse {
		return h.cardinalityLC()
	} else {
		return h.cardinalityNormal()
	}
}

// Uses linear counting to determine the cardinality for the sparse case.
func (h *Hll) cardinalityLC() uint64 {
	if len(h.tempSet) > 0 {
		h.sparseList = merge(h.sparseList, h.tempSet, h.p, h.pPrime)
		h.tempSet = []uint64{}
	}
	return linearCounting(h.mPrime, h.mPrime-h.sparseList.GetNumElements())
}

// Returns the cardinality estimate for the dense case.
func (h *Hll) cardinalityNormal() uint64 {
	inverseSum := float64(0)
	V := uint64(0)

	// calculate the harmonic mean of the values in the registers.
	for i := uint64(0); i < h.m; i++ {
		registerVal := h.M.Get(i)
		inverseSum += 1 / math.Pow(2, float64(registerVal))
		if registerVal == 0 {
			V++
		}
	}
	e1 := h.alpha * float64(h.m*h.m) / inverseSum
	// Take bias into consideration
	var e2 float64
	if e1 <= 5*float64(h.m) {
		e2 = e1 - h.estimateBias(e1)
	} else {
		e2 = e1
	}
	// if not all registers are filled, linear counting is more accurate than the bias-corrected raw estimate.
	var H uint64
	if V != 0 {
		H = linearCounting(h.m, V)
	} else {
		H = roundFloatToUint64(e2)
	}
	if H <= uint64(thresholds[h.p]) { // extracts empirically determined threshold value
		return H
	} else {
		return roundFloatToUint64(e2)
	}
}

// Returns linear counting cardinality estimate.
func linearCounting(m, v uint64) uint64 {
	count := float64(m) * math.Log(float64(m)/float64(v))
	return roundFloatToUint64(count)
}

// Get bias estimation calculated from the empirical results found in appendix.
// If estimate is not in the raw estimates, calculates a weighted mean to determine the bias.
func (h *Hll) estimateBias(e float64) float64 {
	biasData := biasMap[h.p]
	rawEstimate := estimateMap[h.p]
	index := sort.SearchFloat64s(rawEstimate, e)
	if index == len(rawEstimate) {
		return biasData[index-1]
	} else if index == 0 {
		return biasData[0]
	} else {
		weight1 := rawEstimate[index] - e
		weight2 := e - rawEstimate[index-1]
		return (biasData[index]*weight1 + biasData[index-1]*weight2) / (weight1 + weight2)
	}
}

func roundFloatToUint64(value float64) uint64 {
	var round float64
	_, div := math.Modf(value)
	if div >= 0.5 {
		round = math.Ceil(value)
	} else {
		round = math.Floor(value)
	}
	return uint64(round)
}
