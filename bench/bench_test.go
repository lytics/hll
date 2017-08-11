package bench

import (
	"flag"
	"fmt"
	"hash"
	"hash/fnv"
	"math/rand"
	"testing"

	axiom "github.com/axiomhq/hyperloglog"
	clark "github.com/clarkduvall/hyperloglog"
	eclesh "github.com/eclesh/hyperloglog"
	lytics "github.com/lytics/hll"
	fiber "github.com/mynameisfiber/gohll"
	rn "github.com/retailnext/hllpp"
)

var (
	numToAdd = 10000
	verbose  bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")
	flag.Parse()
}

// https://github.com/lytics/hll
func BenchmarkLytics(b *testing.B) {
	b.ReportAllocs()
	r := make(map[uint64]struct{})
	h := lytics.NewHll(14, 20)
	for i := 0; i < numToAdd; i++ {
		val := hash64(randStr(i)).Sum64()
		h.Add(val)
		r[val] = struct{}{}
		if i%10 == 0 {
			h.Cardinality()
		}
	}
	if verbose {
		card := float64(h.Cardinality())
		corr := float64(len(r))
		accuracy := min(card, corr) / max(card, corr)
		b.Logf("[Lytics] cardinality: %v. true: %v. accuracy: %v \n", card, corr, accuracy)
	}
}

// https://github.com/eclesh/hyperloglog
func BenchmarkEclesh(b *testing.B) {
	b.ReportAllocs()
	r := make(map[uint32]struct{})
	h, _ := eclesh.New(1 << 14)
	for i := 0; i < numToAdd; i++ {
		val := hash32(randStr(i)).Sum32()
		h.Add(val)
		r[val] = struct{}{}
		if i%10 == 0 {
			h.Count()
		}
	}
	if verbose {
		card := float64(h.Count())
		corr := float64(len(r))
		accuracy := min(card, corr) / max(card, corr)
		b.Logf("[Eclesh] cardinality: %v. true: %v. accuracy: %v \n", card, corr, accuracy)
	}
}

// https://github.com/clarkduvall/hyperloglog
func BenchmarkClarkDuvall(b *testing.B) {
	b.ReportAllocs()
	r := make(map[uint64]struct{})
	h, _ := clark.NewPlus(14)
	for i := 0; i < numToAdd; i++ {
		val := hash64(randStr(i))
		h.Add(val)
		r[val.Sum64()] = struct{}{}
		if i%10 == 0 {
			h.Count()
		}
	}
	if verbose {
		card := float64(h.Count())
		corr := float64(len(r))
		accuracy := min(card, corr) / max(card, corr)
		b.Logf("[Clark] cardinality: %v. true: %v. accuracy: %v \n", card, corr, accuracy)
	}
}

// https://github.com/retailnext/hllpp
func BenchmarkRetailNext(b *testing.B) {
	b.ReportAllocs()
	r := make(map[uint64]struct{})
	h := rn.New()
	for i := 0; i < numToAdd; i++ {
		val := hash64(randStr(i))
		h.Add(val.Sum(nil))
		r[val.Sum64()] = struct{}{}
		if i%10 == 0 {
			h.Count()
		}
	}
	if verbose {
		card := float64(h.Count())
		corr := float64(len(r))
		accuracy := min(card, corr) / max(card, corr)
		b.Logf("[Retail] cardinality: %v. true: %v. accuracy: %v \n", card, corr, accuracy)
	}
}

// https://github.com/mynameisfiber/gohll
func BenchmarkMyNameIsFiber(b *testing.B) {
	b.ReportAllocs()
	r := make(map[uint64]struct{})
	h, _ := fiber.NewHLL(15)
	h.Hasher = func(s string) uint64 {
		val := hash64(s).Sum64()
		r[val] = struct{}{}
		return val
	}
	for i := 0; i < numToAdd; i++ {
		h.Add(randStr(i))
		if i%10 == 0 {
			h.Cardinality()
		}
	}
	if verbose {
		card := float64(h.Cardinality())
		corr := float64(len(r))
		accuracy := min(card, corr) / max(card, corr)
		b.Logf("[Fiber] cardinality: %v. true: %v. accuracy: %v \n", card, corr, accuracy)
	}
}

// https://github.com/axiomhq/hyperloglog
func BenchmarkAxiomHQ(b *testing.B) {
	b.ReportAllocs()
	r := make(map[uint64]struct{})
	h := axiom.New16()
	for i := 0; i < numToAdd; i++ {
		val := hash64(randStr(i))
		h.Insert(val.Sum(nil))
		r[val.Sum64()] = struct{}{}
		if i%10 == 0 {
			h.Estimate()
		}
	}
	if verbose {
		card := float64(h.Estimate())
		corr := float64(len(r))
		accuracy := min(card, corr) / max(card, corr)
		b.Logf("[Axiom] cardinality: %v. true: %v. accuracy: %v \n", card, corr, accuracy)
	}
}

func hash32(s string) hash.Hash32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h
}

func hash64(s string) hash.Hash64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h
}

func randStr(n int) string {
	i := rand.Uint32()
	return fmt.Sprintf("%s %s", i, n)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
