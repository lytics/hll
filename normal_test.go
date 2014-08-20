package hll

import (
	"os"
	"testing"
)

func TestNormal(t *testing.T) {
	for _, numRegisters := range []uint64{1023, 1024, 1025} { // Try a power of two, also power of two +/- 1.
		iterativeGetSet(t, numRegisters)
	}
}

func TestHuge(t *testing.T) {
	// This test uses an obscene amount of memory and takes a long time. Only run it when requested.
	if len(os.Getenv("HLL_HUGE")) == 0 {
		t.Skip("Skipping gigantic memory test because HLL_HUGE isn't set")
		return
	}

	// 7 billion registers should use over 4GB of memory which seems like a good size to test.
	var numRegisters uint64 = 7 * (1 << 30)
	iterativeGetSet(t, numRegisters)
}

func iterativeGetSet(t *testing.T, numRegisters uint64) {
	cd := newNormal(numRegisters)

	for i := uint64(0); i < numRegisters; i++ {
		valToInsert := uint8(i % 64)
		cd.Set(i, valToInsert)
		readBack := cd.Get(i)
		if readBack != valToInsert {
			t.Fatal(readBack, valToInsert)
		}
	}

	for i := uint64(0); i < numRegisters; i++ {
		readBack := cd.Get(i)
		expected := uint8(i % 64)
		if readBack != expected {
			t.Fatal(readBack, expected)
		}
	}
}
