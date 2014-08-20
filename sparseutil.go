package hll

import (
	"fmt"
	"sort"
)

// Bitstrings are uint64. Rho values (bucket counts in M) are uint64. Indices and p values are uint.
// rho results (position of first 1 in a bitstring) are uint8 because only 6 bits are required to
// encode the position of a bit in a 64-bit sequence (log2(64)==6).

// Return the position of the first set bit, starting with 1. This is the same as the number of
// leading zeros + 1. Returns 63 if no bits were set. Since the result is in [1,63] it can be
// encoded in 6 bits.
func rho(x uint64) uint8 {
	var i uint8
	for i = 0; i < 62 && x&1 == 0; i++ {
		x >>= 1
	}
	return i + 1
}

// x is a hash code.
func encodeHash(x uint64, p, pPrime uint) (hashCode uint64) {
	if x&onesFromTo(64-pPrime, 63-p) == 0 {
		r := rho(extractShift(x, 0, 63-pPrime))
		return concat([]concatInput{
			{x, 64 - pPrime, 63},
			{uint64(r), 0, 5},
			{1, 0, 0}, // this just adds a 1 bit at the end
		})
	} else {
		return concat([]concatInput{
			{x, 64 - pPrime, 63},
			{0, 0, 0}, // this just adds a 0 bit at the end
		})
	}
}

// k is an encoded hash.
func getIndex(k uint64, p, pPrime uint) uint {
	if k&1 == 1 {
		index := uint(extractShift(k, 7, p+6)) // erratum from paper, start index is 7, not 6
		return index
	} else {
		index := uint(extractShift(k, 1, p)) // erratum from paper, end index is p, not p+1
		return index
	}
}

// k is an encoded hash.
func decodeHash(k uint64, p, pPrime uint) (idx uint, rhoW uint8) {
	var r uint8
	if k&1 == 1 {
		r = uint8(extractShift(k, 1, 6) + uint64(pPrime-p))
	} else {
		r = rho(extractShift(k, 1, pPrime-p-1))
	}
	return getIndex(k, p, pPrime), r
}

type mergeElem struct {
	valid   bool
	index   uint
	rho     uint8
	encoded uint64
}

// tmpSet must be sorted before calling this function. TODO maybe sorting could happen in here?
func merge(cs *sparse, tmpSet []uint64, p, pPrime uint) *sparse {
	// sorts tmpSet by index, and not by raw integer value
	sortHashcodesByIndex(tmpSet, p, pPrime)

	var sparseElem, tmpElem mergeElem

	sparseIter := cs.GetIterator()
	loadSparseElem := func() {
		sparseElem.encoded, sparseElem.valid = sparseIter()
		if sparseElem.valid {
			sparseElem.index, sparseElem.rho = decodeHash(sparseElem.encoded, p, pPrime)
		}
	}

	tmpSetOffset := 0
	loadTmpSetElem := func() {
		// temp set could contain multiple elements with the same index value
		for {
			if tmpSetOffset >= len(tmpSet) {
				tmpElem.valid = false
				return
			}
			lastIndex := tmpElem.index
			tmpElem.encoded = tmpSet[tmpSetOffset]
			tmpElem.index, tmpElem.rho = decodeHash(tmpElem.encoded, p, pPrime)
			tmpElem.valid = true
			tmpSetOffset++
			if tmpSetOffset > 0 && tmpElem.index == lastIndex {
				continue
			}
			return
		}
	}

	loadSparseElem()
	loadTmpSetElem()

	output := newSparse(cs.GetNumElements())

	for sparseElem.valid && tmpElem.valid {
		var toAppend uint64
		if sparseElem.index < tmpElem.index {
			toAppend = sparseElem.encoded
			loadSparseElem()
		} else if tmpElem.index < sparseElem.index {
			toAppend = tmpElem.encoded
			loadTmpSetElem()
		} else { // The indexes are equal. Keep the one with the highest rho value.
			if sparseElem.rho > tmpElem.rho {
				toAppend = sparseElem.encoded
			} else {
				toAppend = tmpElem.encoded
			}
			loadTmpSetElem()
			loadSparseElem()
		}
		output.Add(toAppend)
	}

	for tmpElem.valid {
		output.Add(tmpElem.encoded)
		loadTmpSetElem()
	}

	for sparseElem.valid {
		output.Add(sparseElem.encoded)
		loadSparseElem()
	}

	return output
}

func toNormal(s *sparse, p, pPrime uint) normal {
	m := 1 << p
	M := newNormal(uint64(m))

	it := s.GetIterator()
	for {
		k, ok := it()
		if !ok {
			break
		}
		idx, r := decodeHash(k, p, pPrime)
		val := maxU8(M.Get(uint64(idx)), r)
		M.Set(uint64(idx), val)
	}
	return M
}

func maxU8(x, y uint8) uint8 {
	if x >= y {
		return x
	}
	return y
}

// For debugging purposes, return the input as "binary/hex/decimal"
func binU(x uint) string {
	return bin(uint64(x))
}

// For debugging purposes, return the input as "binary/hex/decimal"
func bin(x uint64) string {
	s := fmt.Sprintf("/%016x/%d", x, x)
	for i := 0; i < 64; i++ {
		thisBit := "0"
		if x&1 == 1 {
			thisBit = "1"
		}
		s = thisBit + s
		x >>= 1
	}

	return s
}

func sortHashcodesByIndex(xs []uint64, p, pPrime uint) {
	sort.Sort(uint64Sorter{xs, p, pPrime})
}

type uint64Sorter struct {
	xs        []uint64
	p, pPrime uint
}

func (u uint64Sorter) Len() int {
	return len(u.xs)
}

func (u uint64Sorter) Less(i, j int) bool {
	iIndex := getIndex(u.xs[i], u.p, u.pPrime)
	jIndex := getIndex(u.xs[j], u.p, u.pPrime)
	if iIndex != jIndex {
		return iIndex < jIndex
	}

	// When two elements have the same index, sort in descending order of rho. This means that the
	// highest rho value will be seen first, and subsequent elements can be discarded whem merging.
	_, iRho := decodeHash(u.xs[i], u.p, u.pPrime)
	_, jRho := decodeHash(u.xs[j], u.p, u.pPrime)
	return iRho > jRho
}

func (u uint64Sorter) Swap(i, j int) {
	tmp := u.xs[i]
	u.xs[i] = u.xs[j]
	u.xs[j] = tmp
}
