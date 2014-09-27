package hll

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

const (
	alpha_16 = 0.673
	alpha_32 = 0.697
	alpha_64 = 0.709
)

type Hll struct {
	bigM                normal   // M is used for the dense case, and registers the rho values for each hashed index.
	sparseList          *sparse  // This will be nil if isSparse==false. Used for sparse case for aggregation
	tempSet             []uint64 // used to store values temporarilty for the sparse case
	alpha               float64  // constant used in cardinality calculation
	isSparse            bool     // boolean flag that determines when to switch over to the dense case
	p, pPrime           uint     // precision bits for dense and sparse cases
	m, mPrime           uint64   // register sizes for dense and sparse cases
	mergeSizeBits       uint64   // the limit for the size of the temp set
	sparseThresholdBits uint64   // the limit for the size of the sparseList, indicates when to switch to dense.
}

// Initialize a new hyper-log-log struct based on inputs p and p'.
// Google recommends that p be set to 14, and p' to equal either 20 or 25.
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

	h.sparseList = newSparse(0)
	h.tempSet = []uint64{}

	// The sparse threshold (the threshold for when to convert to the normal case) is set to m.6 bits.
	h.sparseThresholdBits = h.m * 6

	// When the temp set reaches 25% of the maximum size for sparse register storage, merge the
	// temp set with the sparse list.
	h.mergeSizeBits = h.sparseThresholdBits / 4

	return h
}

// Add takes a hash and updates the cardinality estimation data structures.
//
// The input should be a hash of whatever type you're estimating of. For example, if you're
// estimating the cardinality of a stream of strings, you'd pass the hash of each string to this
// function.
func (h *Hll) Add(x uint64) {
	if h.isSparse {
		h.addSparse(x)
	} else {
		h.addNormal(x)
	}
}

// Combine() merges two HyperLogLog++ calculations. This allows you to parallelize cardinality
// estimation: each thread can process a shard of the input, then the results can be merged later to
// give the cardinality of the entire data set (the union of the shards).
//
// WARNING: The "other" parameter may be mutated during this call! It may be converted from a sparse
// to dense representation, which may affect its space usage and precision. This is a deliberate
// design decision that helps to minimize memory consumption.
//
// The inputs must have the same p and pPrime or this function will panic.
// The Google paper doesn't give an algorithm for this operation, but its existence is implied, and
// the ability to do this combine operation is one of the main benefits of using a HyperLogLog-type
// algorithm in the first place.
func (h *Hll) Combine(other *Hll) {
	if h.p != other.p || h.pPrime != other.pPrime {
		panic(fmt.Sprintf("Parameter mismatch: p=%d/%d, pPrime=%d/%d", h.p, other.p, h.pPrime,
			other.pPrime))
	}

	other.mergeTmpSetIfAny()

	// If the other Hll is normal (not sparse), then the union will be normal. If this Hll isn't
	// also normal, do the conversion now.
	if h.isSparse && !other.isSparse {
		h.switchToNormal()
	}

	if h.isSparse && other.isSparse { // Case 1: both inputs are sparse
		capBytes := maxU64(h.sparseList.SizeInBytes(), other.sparseList.SizeInBytes())
		h.sparseList = merge(h.p, h.pPrime, capBytes, h.sparseList.GetIterator(),
			other.sparseList.GetIterator())
		if h.sparseList.SizeInBits() > h.sparseThresholdBits {
			h.switchToNormal()
		}
	} else if !h.isSparse && !other.isSparse { // Case 2: both inputs are normal
		for i := uint64(0); i < h.m; i++ {
			h.bigM.Set(i, maxU8(h.bigM.Get(i), other.bigM.Get(i)))
		}
	} else { // Case 3: h is normal, other is sparse
		otherIt := other.sparseList.GetIterator()
		for {
			hashCode, ok := otherIt()
			if !ok {
				break
			}
			index, r := decodeHash(hashCode, h.p, h.pPrime)
			h.bigM.Set(index, maxU8(h.bigM.Get(index), r))
		}
	}
}

func (h *Hll) addSparse(x uint64) {
	k := encodeHash(x, h.p, h.pPrime)
	h.tempSet = append(h.tempSet, k)

	tempSetBits := uint64(len(h.tempSet)) * 64
	if tempSetBits > h.mergeSizeBits {
		h.mergeTmpSetIfAny()
	}
}

func (h *Hll) mergeTmpSetIfAny() {
	if !h.isSparse || len(h.tempSet) == 0 {
		return
	}
	sortHashcodesByIndex(h.tempSet, h.p, h.pPrime)
	tmpSetIt := makeU64SliceIt(h.tempSet)
	sparseIt := h.sparseList.GetIterator()
	h.sparseList = merge(h.p, h.pPrime, h.sparseList.SizeInBytes(), sparseIt, tmpSetIt)
	h.tempSet = []uint64{}
	if h.sparseList.SizeInBits() > h.sparseThresholdBits {
		h.switchToNormal()
	}
}

func (h *Hll) switchToNormal() {
	h.isSparse = false
	h.bigM = toNormal(h.sparseList, h.p, h.pPrime)
	h.sparseList = nil
}

func (h *Hll) addNormal(x uint64) {
	offset := (64 - h.p)
	idx := x >> offset
	r := rho(x)
	if r > h.bigM.Get(idx) {
		h.bigM.Set(idx, r)
	}
}

// Returns the estimated cardinality (the number of unique inputs seen so far).
func (h *Hll) Cardinality() uint64 {
	// This step doesn't appear in the upstream paper, but it allows us to interleave adding new
	// inputs with cardinality calculations. If we didn't do this step, there's a subtle edge case
	// where the sparse list could grow without being converted into the dense representation.
	h.mergeTmpSetIfAny()

	if h.isSparse {
		return h.cardinalityLC()
	} else {
		return h.cardinalityNormal()
	}
}

// Uses linear counting to determine the cardinality for the sparse case.
func (h *Hll) cardinalityLC() uint64 {
	return linearCounting(h.mPrime, h.mPrime-h.sparseList.GetNumElements())
}

// Returns the cardinality estimate for the dense case.
func (h *Hll) cardinalityNormal() uint64 {
	inverseSum := float64(0)
	V := uint64(0)

	// calculate the harmonic mean of the values in the registers.
	for i := uint64(0); i < h.m; i++ {
		registerVal := h.bigM.Get(i)
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

// When marshalling an Hll to JSON, we only marshal a subset of its fields.
type jsonableHll struct {
	BigM       *normal `json:"M,omitempty"`
	SparseList *sparse `json:"s,omitempty"`
	P          uint    `json:"p"`
	PPrime     uint    `json:"pp"`
}

// Convert the Hll struct into JSON.
func (h *Hll) MarshalJSON() ([]byte, error) {
	// Combine tmpSet with sparse list. This saves serializing the tmpSet, which saves space.
	h.mergeTmpSetIfAny()

	bigM := &h.bigM
	if len(*bigM) == 0 {
		bigM = nil
	}

	return json.Marshal(&jsonableHll{bigM, h.sparseList, h.p, h.pPrime})
}

// Unmarshals JSON byte-array into a Hll struct.
func (h *Hll) UnmarshalJSON(buf []byte) error {
	j := jsonableHll{}

	if err := json.Unmarshal(buf, &j); err != nil {
		return err
	}

	// Copy field values from the jsonable model to the real Hll struct.
	*h = *NewHll(j.P, j.PPrime)
	h.sparseList = nil
	h.bigM = nil

	h.sparseList = j.SparseList
	if j.BigM != nil {
		h.bigM = *j.BigM
	}
	h.isSparse = (h.sparseList != nil)
	return nil
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
