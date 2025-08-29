package shmap

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---- Baseline Implementations --------------------------------------------

type rwMap struct {
	mu sync.RWMutex
	m  map[string]int
}

func newRWMap() *rwMap { return &rwMap{m: make(map[string]int)} }

func (r *rwMap) Get(k string) (int, bool) {
	r.mu.RLock()
	v, ok := r.m[k]
	r.mu.RUnlock()
	return v, ok
}
func (r *rwMap) Set(k string, v int) { r.mu.Lock(); r.m[k] = v; r.mu.Unlock() }
func (r *rwMap) Delete(k string)     { r.mu.Lock(); delete(r.m, k); r.mu.Unlock() }
func (r *rwMap) Range(fn func(string, int) bool) {
	r.mu.RLock()
	for k, v := range r.m {
		if !fn(k, v) {
			break
		}
	}
	r.mu.RUnlock()
}

type syncMap struct{ m sync.Map }

func newSyncMap() *syncMap { return &syncMap{} }

func (s *syncMap) Get(k string) (int, bool) {
	v, ok := s.m.Load(k)
	if !ok {
		return 0, false
	}
	return v.(int), true
}
func (s *syncMap) Set(k string, v int) { s.m.Store(k, v) }
func (s *syncMap) Delete(k string)     { s.m.Delete(k) }
func (s *syncMap) Range(fn func(string, int) bool) {
	s.m.Range(func(k, v any) bool { return fn(k.(string), v.(int)) })
}

// ---- Benchmark Utilities -------------------------------------------------

const benchKeyCount = 100_000

func genKeys(n int) []string {
	keys := make([]string, n)
	for i := range n {
		keys[i] = "key-" + strconv.Itoa(i)
	}
	return keys
}

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func prepareShmap(shards int) *Map[string, int] {
	m := WithShards[string, int](shards)
	keys := genKeys(benchKeyCount)
	for i, k := range keys {
		m.Set(k, i)
	}
	return m
}

func prepareRWMap() *rwMap {
	r := newRWMap()
	keys := genKeys(benchKeyCount)
	for i, k := range keys {
		r.Set(k, i)
	}
	return r
}

func prepareSyncMap() *syncMap {
	s := newSyncMap()
	keys := genKeys(benchKeyCount)
	for i, k := range keys {
		s.Set(k, i)
	}
	return s
}

// ---- Core Performance Benchmarks -----------------------------------------

func BenchmarkReadMostly(b *testing.B) {
	const writePct = 10 // 10% writes, 90% reads
	keys := genKeys(benchKeyCount)

	b.Run("shmap", func(b *testing.B) {
		m := prepareShmap(128)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < writePct {
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})

	b.Run("rwmap", func(b *testing.B) {
		m := prepareRWMap()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < writePct {
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})

	b.Run("syncmap", func(b *testing.B) {
		m := prepareSyncMap()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < writePct {
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})
}

func BenchmarkWriteHeavy(b *testing.B) {
	const writePct = 50 // 50% writes, 50% reads
	keys := genKeys(benchKeyCount)

	b.Run("shmap", func(b *testing.B) {
		m := prepareShmap(128)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < writePct {
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})

	b.Run("rwmap", func(b *testing.B) {
		m := prepareRWMap()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < writePct {
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})

	b.Run("syncmap", func(b *testing.B) {
		m := prepareSyncMap()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < writePct {
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})
}

// ---- Contention & Scaling Benchmarks -------------------------------------

func BenchmarkGoroutineScaling(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8, 16, 32}
	keys := genKeys(1000) // Smaller key set for more contention

	for _, count := range goroutineCounts {
		b.Run(fmt.Sprintf("%dgoroutines", count), func(b *testing.B) {
			maps := map[string]interface {
				Get(string) (int, bool)
				Set(string, int)
			}{
				"shmap":   prepareShmap(128),
				"rwmap":   prepareRWMap(),
				"syncmap": prepareSyncMap(),
			}

			for name, m := range maps {
				b.Run(name, func(b *testing.B) {
					b.ResetTimer()

					var wg sync.WaitGroup
					opsPerGoroutine := b.N / count

					for i := range count {
						wg.Add(1)
						go func(seed int64) {
							defer wg.Done()
							r := rand.New(rand.NewSource(seed))
							for _ = range opsPerGoroutine {
								k := keys[r.Intn(len(keys))]
								if r.Intn(100) < 20 { // 20% writes
									m.Set(k, r.Int())
								} else {
									_, _ = m.Get(k)
								}
							}
						}(int64(i))
					}
					wg.Wait()
				})
			}
		})
	}
}

func BenchmarkHighContention(b *testing.B) {
	// Single key hit by all goroutines - worst case for sharding
	const hotKey = "contention-key"

	b.Run("shmap", func(b *testing.B) {
		m := WithShards[string, int](128)
		m.Set(hotKey, 42)
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				if r.Intn(100) < 10 { // 10% writes
					m.Set(hotKey, r.Int())
				} else {
					_, _ = m.Get(hotKey)
				}
			}
		})
	})

	b.Run("syncmap", func(b *testing.B) {
		m := prepareSyncMap()
		m.Set(hotKey, 42)
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				if r.Intn(100) < 10 { // 10% writes
					m.Set(hotKey, r.Int())
				} else {
					_, _ = m.Get(hotKey)
				}
			}
		})
	})
}

// ---- Shard Scaling Benchmarks --------------------------------------------

func BenchmarkShardScaling(b *testing.B) {
	shardCounts := []int{16, 64, 128, 256, 512}
	keys := genKeys(benchKeyCount)

	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("%dshards", shards), func(b *testing.B) {
			m := prepareShmap(shards)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				r := newRand()
				for pb.Next() {
					k := keys[r.Intn(len(keys))]
					if r.Intn(100) < 10 { // 10% writes
						m.Set(k, r.Int())
					} else {
						_, _ = m.Get(k)
					}
				}
			})
		})
	}
}

// ---- Zero-Allocation Benchmarks ------------------------------------------

func BenchmarkHashFunctions(b *testing.B) {
	testString := "benchmark-test-key-with-decent-length"
	testInt := int64(1234567890123456)
	testFloat := float64(123.456789)

	b.Run("StringHasher", func(b *testing.B) {
		b.ReportAllocs()
		for _ = range b.N {
			_ = StringHasher(testString)
		}
	})

	b.Run("IntHasher", func(b *testing.B) {
		b.ReportAllocs()
		for _ = range b.N {
			_ = IntHasher(testInt)
		}
	})

	b.Run("FloatHasher", func(b *testing.B) {
		b.ReportAllocs()
		for _ = range b.N {
			_ = FloatHasher(testFloat)
		}
	})
}

func BenchmarkOperationalAllocations(b *testing.B) {
	keys := genKeys(benchKeyCount)

	b.Run("shmap_operations", func(b *testing.B) {
		m := prepareShmap(128)
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < 10 { // 10% writes
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})

	b.Run("syncmap_operations", func(b *testing.B) {
		m := prepareSyncMap()
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			r := newRand()
			for pb.Next() {
				k := keys[r.Intn(len(keys))]
				if r.Intn(100) < 10 { // 10% writes
					m.Set(k, r.Int())
				} else {
					_, _ = m.Get(k)
				}
			}
		})
	})
}

// ---- Workload Pattern Benchmarks -----------------------------------------

func BenchmarkWorkloadPatterns(b *testing.B) {
	patterns := []struct {
		name     string
		writePct int
	}{
		{"Read95Write5", 5},
		{"Read90Write10", 10},
		{"Read80Write20", 20},
		{"Read70Write30", 30},
		{"Read50Write50", 50},
	}

	keys := genKeys(benchKeyCount)

	for _, pattern := range patterns {
		b.Run(pattern.name, func(b *testing.B) {
			maps := map[string]interface {
				Get(string) (int, bool)
				Set(string, int)
			}{
				"shmap":   prepareShmap(128),
				"syncmap": prepareSyncMap(),
			}

			for name, m := range maps {
				b.Run(name, func(b *testing.B) {
					b.ResetTimer()
					b.RunParallel(func(pb *testing.PB) {
						r := newRand()
						for pb.Next() {
							k := keys[r.Intn(len(keys))]
							if r.Intn(100) < pattern.writePct {
								m.Set(k, r.Int())
							} else {
								_, _ = m.Get(k)
							}
						}
					})
				})
			}
		})
	}
}

// ---- Range Operations Benchmark ------------------------------------------

func BenchmarkRangeOperations(b *testing.B) {
	const rangeKeyCount = 10000

	// Pre-populate with additional keys for range testing
	populateForRange := func(m interface{ Set(string, int) }) {
		for i := range rangeKeyCount {
			m.Set(fmt.Sprintf("range-key-%d", i), i)
		}
	}

	b.Run("shmap", func(b *testing.B) {
		m := prepareShmap(128)
		populateForRange(m)
		b.ResetTimer()

		for _ = range b.N {
			count := 0
			m.Range(func(string, int) bool {
				count++
				return true
			})
			runtime.KeepAlive(count)
		}
	})

	b.Run("rwmap", func(b *testing.B) {
		m := prepareRWMap()
		populateForRange(m)
		b.ResetTimer()

		for _ = range b.N {
			count := 0
			m.Range(func(string, int) bool {
				count++
				return true
			})
			runtime.KeepAlive(count)
		}
	})

	b.Run("syncmap", func(b *testing.B) {
		m := prepareSyncMap()
		populateForRange(m)
		b.ResetTimer()

		for _ = range b.N {
			count := 0
			m.Range(func(string, int) bool {
				count++
				return true
			})
			runtime.KeepAlive(count)
		}
	})
}

// ---- Memory & Construction Benchmarks ------------------------------------

func BenchmarkConstructionCost(b *testing.B) {
	b.Run("shmap", func(b *testing.B) {
		b.ReportAllocs()
		for _ = range b.N {
			m := WithShards[string, int](128)
			runtime.KeepAlive(m)
		}
	})

	b.Run("rwmap", func(b *testing.B) {
		b.ReportAllocs()
		for _ = range b.N {
			m := newRWMap()
			runtime.KeepAlive(m)
		}
	})

	b.Run("syncmap", func(b *testing.B) {
		b.ReportAllocs()
		for _ = range b.N {
			m := newSyncMap()
			runtime.KeepAlive(m)
		}
	})
}

// ---- Key Type Benchmarks ------------------------------------------------

func BenchmarkKeyTypes(b *testing.B) {
	const iterations = 100000

	b.Run("IntKeys", func(b *testing.B) {
		m := WithShards[int, string](128)
		for i := 0; i < iterations; i++ {
			m.Set(i, fmt.Sprintf("value-%d", i))
		}
		b.ReportAllocs()
		b.ResetTimer()

		for i := range b.N {
			_, _ = m.Get(i % iterations)
		}
	})

	b.Run("StringKeys", func(b *testing.B) {
		m := WithShards[string, int](128)
		keys := genKeys(iterations)
		for i, k := range keys {
			m.Set(k, i)
		}
		b.ReportAllocs()
		b.ResetTimer()

		for i := range b.N {
			_, _ = m.Get(keys[i%len(keys)])
		}
	})
}

// ---- Utility Benchmarks --------------------------------------------------

func BenchmarkAtomicLoadOverhead(b *testing.B) {
	var v atomic.Value
	v.Store(42)
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = v.Load()
		}
	})
}
