package bench

import (
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

// https://github.com/lytics/hll
func BenchmarkLytics(b *testing.B) {
	b.ReportAllocs()
	h := lytics.NewHll(14, 20)
	for i := 0; i < b.N; i++ {
		h.Add(hash64(randStr(i)).Sum64())
		h.Cardinality()
	}
}

// https://github.com/eclesh/hyperloglog
func BenchmarkEclesh(b *testing.B) {
	b.ReportAllocs()
	h, _ := eclesh.New(1 << 14)
	for i := 0; i < b.N; i++ {
		h.Add(hash32(randStr(i)).Sum32())
		h.Count()
	}
}

// https://github.com/clarkduvall/hyperloglog
func BenchmarkClarkDuvall(b *testing.B) {
	b.ReportAllocs()
	h, _ := clark.NewPlus(14)
	for i := 0; i < b.N; i++ {
		h.Add(hash64(randStr(i)))
		h.Count()
	}
}

// https://github.com/retailnext/hllpp
func BenchmarkRetailNext(b *testing.B) {
	b.ReportAllocs()
	h := rn.New()
	for i := 0; i < b.N; i++ {
		h.Add(hash64(randStr(i)).Sum(nil))
		h.Count()
	}
}

// https://github.com/mynameisfiber/gohll
func BenchmarkMyNameIsFiber(b *testing.B) {
	b.ReportAllocs()
	h, _ := fiber.NewHLL(15)
	h.Hasher = func(s string) uint64 {
		return hash64(s).Sum64()
	}
	for i := 0; i < b.N; i++ {
		h.Add(randStr(i))
		h.Cardinality()
	}
}

// https://github.com/axiomhq/hyperloglog
func BenchmarkAxiomHQ(b *testing.B) {
	b.ReportAllocs()
	h := axiom.New16()
	for i := 0; i < b.N; i++ {
		h.Insert(hash64(randStr(i)).Sum(nil))
		h.Estimate()
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
