package hll

import (
	"fmt"
)

type normal []byte

func newNormal(numRegisters uint64) normal {
	// We can store 4 6-bit registers in 3 bytes (4 * 6 == 3 * 8)
	numBytes := (numRegisters*3)/4 + 1 // +1 to round up
	return make([]byte, numBytes)
}

// This function assumes that registerIdx is within range. It may panic if not.
func (n normal) Get(registerIdx uint64) uint8 {
	byteIdx, startBit, numInSecondByte := bitPosn(registerIdx)

	result := (n[byteIdx] >> startBit) & 0x3f
	if numInSecondByte == 0 {
		return result
	}
	result <<= numInSecondByte
	lowOrderMask := uint8(onesFromTo(0, numInSecondByte-1))
	result |= n[byteIdx+1] & lowOrderMask
	return result
}

func (n normal) Set(registerIdx uint64, val uint8) {
	byteIdx, startBit, numInSecondByte := bitPosn(registerIdx)

	// if val&0x3f != val {
	// 	panic("register values should only have their lower 6 bits set.") // TODO remove for prod
	// }

	b1 := n[byteIdx]
	b1 = b1 &^ uint8(onesFromTo(startBit, startBit+6-1)) // Clear bits holding this register.
	b1 |= (val >> numInSecondByte) << startBit
	n[byteIdx] = b1

	if numInSecondByte == 0 {
		return
	}

	b2 := n[byteIdx+1]
	lowOrderMask := uint8(onesFromTo(0, numInSecondByte-1))
	b2 = b2 &^ lowOrderMask // Clear bits holding this register.
	b2 |= (val & lowOrderMask)
	n[byteIdx+1] = b2
}

func (n normal) Size() int {
	return len(n) / 6
}

func (n *normal) MarshalJSON() ([]byte, error) {
	compressed, err := snappyB64(*n)
	if err != nil {
		return nil, err
	}

	// Wrap the base64 in quotes so it's a valid JSON string.
	buf := make([]byte, len(compressed)+2)
	buf[0] = '"'
	copy(buf[1:], compressed)
	buf[len(buf)-1] = '"'

	return buf, nil
}

func (n *normal) UnmarshalJSON(buf []byte) error {
	if len(buf) < 2 {
		return fmt.Errorf("A marshaled \"normal\" should be at least two bytes, including quotes")
	}
	buf = buf[1 : len(buf)-1] // Remove the quotes from the JSON string

	uncompressed, err := unsnappyB64(buf)
	if err != nil {
		return err
	}

	*n = uncompressed
	return nil
}

// Given a register number, returns the bit position where it can be found in the byte slice.
func bitPosn(registerIdx uint64) (byteIdx uint64, startBit, numInSecondByte uint) {
	bitIdx := registerIdx * 6

	byteIdx = bitIdx / 8
	startBit = uint(bitIdx % 8)
	numInFirstByte := minUint(6, 8-startBit)
	numInSecondByte = 6 - numInFirstByte

	return
}

func minUint(x, y uint) uint {
	if x <= y {
		return x
	}
	return y
}
