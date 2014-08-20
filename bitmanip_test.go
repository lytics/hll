package hll

import "testing"

func TestOnesTo(t *testing.T) {
	testCases := []struct {
		startPos, endPos uint
		expectResult     uint64
	}{
		{0, 0, 1},
		{63, 63, 1 << 63},
		{2, 4, 4 + 8 + 16},
		{56, 63, 0xFF00000000000000},
	}

	for i, testCase := range testCases {
		actualResult := onesFromTo(testCase.startPos, testCase.endPos)
		if testCase.expectResult != actualResult {
			t.Errorf("Case %d actual result was %v", i, actualResult)
		}
	}
}

func TestExtractShift(t *testing.T) {
	testCases := []struct {
		input            uint64
		startPos, endPos uint
		expectResult     uint64
	}{
		{0, 0, 63, 0},
		{0xAABBCCDD00, 8, 47, 0xAABBCCDD},
		{0xFF00000000000000, 56, 63, 0xFF},
		{0xFF, 0, 7, 0xFF},
	}

	for i, testCase := range testCases {
		actualResult := extractShift(testCase.input, testCase.startPos, testCase.endPos)
		if testCase.expectResult != actualResult {
			t.Errorf("Case %d actual result was %v", i, actualResult)
		}
	}
}

func TestConcat(t *testing.T) {
	testCases := []struct {
		inputs       []concatInput
		expectResult uint64
	}{
		{[]concatInput{{0xABCD, 0, 15}, {0x1234, 0, 15}}, 0xABCD1234},
		{[]concatInput{{0x0000ABCD0000, 16, 31}, {0x1234000000000000, 48, 63}}, 0xABCD1234},
		{[]concatInput{{0x1234, 0, 15}, {0x12, 0, 7}}, 0x123412},
	}

	for i, testCase := range testCases {
		actualResult := concat(testCase.inputs)
		if testCase.expectResult != actualResult {
			t.Errorf("Case %d actual result was %x", i, actualResult)
		}
	}
}
