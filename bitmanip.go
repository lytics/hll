package hll

// Bit manipulation functions

const all1s uint64 = 1<<64 - 1

// Return a bitmask containing ones from position startPos to endPos, inclusive.
// startPos and endPos are 0-indexed so they should be in [0,63].
// startPos should be less than or equal to endPos.
func onesFromTo(startPos, endPos uint) uint64 {
	// if endPos < startPos {
	// 	panic("assert")
	// }

	// Generate two overlapping sequences of 1s, and keep the overlap.
	highOrderOnes := all1s << startPos
	lowOrderOnes := all1s >> (64 - endPos - 1)
	result := highOrderOnes & lowOrderOnes
	return result
}

// Return bits x[startPos:endPos] inclusive, shifted into the low order bits of the result.
// startPos and endPos are 0-indexed so they should be in [0,63].
// startPos should be less than or equal to endPos.
func extractShift(x uint64, startPos, endPos uint) uint64 {
	mask := onesFromTo(startPos, endPos)
	bits := x & mask
	placed := bits >> startPos
	// fmt.Printf("extractShift length %d returning %d\n", endPos-startPos+1, placed)
	return placed
}

// startPos and endPos are inclusive.
// startPos and endPos are 0-indexed so they should be in [0,63].
// startPos should be less than or equal to endPos.
type concatInput struct {
	x                uint64
	startPos, endPos uint
}

// Extract bit strings from multiple inputs and concatenate them input a single bitstring. Each
// input is a bitstring and a range to be taken from that bitstring.
func concat(inputs []concatInput) uint64 {
	var accum uint64 = 0
	for _, input := range inputs {
		inputNumBits := input.endPos - input.startPos + 1
		accum <<= inputNumBits
		accum |= extractShift(input.x, input.startPos, input.endPos)
	}
	return accum
}
