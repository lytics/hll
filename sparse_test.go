package hll

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestSparse(t *testing.T) {
	cs := newSparse(3)
	inputs := []uint64{5, 7, 3, 4, 7, 2, 3, 4, 1}
	for _, x := range inputs {
		cs.Add(x)
	}

	iter := cs.GetIterator()

	for i := 0; i < len(inputs); i++ {
		output, ok := iter()
		assert.T(t, ok)
		assert.Equal(t, output, inputs[i])
	}

	_, ok := iter()
	assert.T(t, !ok)
}
