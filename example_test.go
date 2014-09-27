package hll

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// A simple walkthrough on how to use Hll.
func Example() {
	const (
		p           = 14 // Max memory usage is 0.75 * 2^p bytes
		pPrime      = 25 // Setting this is a bit more complicated, Google recommends 25.
		numToInsert = 1000000
	)

	// You can use any good hash function, truncated to 8 bytes to fit in a uint64.
	hashU64 := func(s string) uint64 {
		sha1Hash := sha1.Sum([]byte(s))
		return binary.LittleEndian.Uint64(sha1Hash[0:8])
	}

	hll := NewHll(p, pPrime)

	// For this example, our inputs will just be strings, e.g. "1", "2"
	for i := 0; i < numToInsert; i++ {
		inputString := strconv.Itoa(i)

		// To use HLL, you hash your item, convert the hash to uint64, and pass it to Add().
		hll.Add(hashU64(inputString))
	}

	// Duplicates do not affect the cardinality. The following loop has no effect.
	for i := 0; i < 10000; i++ {
		hll.Add(hashU64("1"))
	}

	// We inserted 1M unique elements, the cardinality should be roughly 1M.
	fmt.Printf("%d\n", hll.Cardinality())
	// Output: 989546
}

// This example builds off of the idea of statistical bootstrapping.
// Bootstrapping is a resampling technique, where the basic idea is to randomly sample with
// replacement from a data set to create a new data set of the same size.
// Theoretically, bootstrapping guarantees that around 63.2% of the observations will be unique.
// Using the Law of Large Number, and our HyperLogLog++ algorithm, we can efficiently estimate the cardinality of
// the bootstrapped sets, and see if the percentage of unique observations converges to the theoretical value of 63.2%.
// We can also merge together all of the Hll estimators, to see if the combined estimator can accurately estimate
// the size of the set.
// For more informatoin about boostrapping, see: http://www.cs.berkeley.edu/~ameet/blb_workshop_slides.pdf
func Example_bootstrap() {
	const (
		numToInsert = 100000 // the size of each boostrapped sample.
		n           = 25     // the number of boostrapped samples.
		p           = 14     // Max memory usage is 0.75 * 2^p bytes
		pPrime      = 25     // Google Recommends setting p' to 25.
	)

	// You can use any good hash function, truncated to 8 bytes to fit in a uint64.
	hashU64 := func(s string) uint64 {
		sha1Hash := sha1.Sum([]byte(s))
		return binary.LittleEndian.Uint64(sha1Hash[0:8])
	}

	// Random sampling with replacement.
	sampleBootstrap := func(numTotal int) []string {
		bootstrap := make([]string, numTotal)
		for i := 0; i < numTotal; i++ {
			rand.Seed(time.Now().UnixNano())
			// For this example, our inputs will just be strings, e.g. "47", "2"
			bootstrap[i] = strconv.Itoa(rand.Intn(numTotal))
		}
		return bootstrap
	}

	// For calculating the average of uint64 values.
	mean := func(values []uint64) float64 {
		sum := float64(0)
		for _, val := range values {
			sum += float64(val)
		}
		return sum / float64(len(values))
	}

	// create the Hll cardinality estimators for the boostrapped samples.
	bootstraps := make([]*Hll, n)

	for i := 0; i < n; i++ {
		hll := NewHll(p, pPrime)
		samp := sampleBootstrap(numToInsert)
		for _, val := range samp {
			hll.Add(hashU64(val))
		}
		bootstraps[i] = hll
	}

	// An empty Hll estimator, to be merged
	mergedEstimated := NewHll(p, pPrime)

	estimates := make([]uint64, n)
	for idx, hll := range bootstraps {
		estimates[idx] = hll.Cardinality()
		mergedEstimated.Combine(hll)
	}

	// The average percent of unique observations. Ideally, this number should be close to 63.2%
	fmt.Printf("Average Percentage of Unique Observations: %v \n", mean(estimates)/numToInsert)
	// Output: 64.413

	// The cardinality estimate from the merged Hll estimators. Ideally, this should be close to 100000
	fmt.Printf("Cardinality Estimate from Merged Hll's: %v \n", mergedEstimated.Cardinality())
	// Output: 115537
}
