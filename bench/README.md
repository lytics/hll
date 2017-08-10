Golang Hyperloglog Benchmark
====

Results from benchmarks

```bash
$ go test -v -bench=. 
BenchmarkLytics-8          	   10000	    118133 ns/op	    3199 B/op	       8 allocs/op
BenchmarkEclesh-8          	   10000	    208808 ns/op	     113 B/op	       5 allocs/op
BenchmarkClarkDuvall-8     	   10000	    171107 ns/op	   21831 B/op	      13 allocs/op
BenchmarkRetailNext-8      	   10000	    172181 ns/op	   19727 B/op	      17 allocs/op
BenchmarkMyNameIsFiber-8   	   10000	    426883 ns/op	     175 B/op	       5 allocs/op
BenchmarkAxiomHQ-8         	   10000	    169566 ns/op	   21967 B/op	      14 allocs/op
PASS
```