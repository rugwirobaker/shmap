# shmap

A high-performance, lock-sharded hash map for Go with zero-allocation hashing and generic type safety.

## Overview

`shmap` was designed to solve contention bottlenecks in high-concurrency scenarios where multiple goroutines frequently access shared map data. By sharding locks across multiple map segments, `shmap` dramatically reduces lock contention while maintaining the familiar map interface you expect.

**What started as a focused solution for concurrency became a versatile, type-safe generic map that handles most use cases with excellent performance characteristics.**

## Key Features

- **Reduced Lock Contention**: Configurable lock sharding distributes concurrent access across multiple segments
- **Zero-Allocation Hashing**: Custom hash functions for common types (string, int, uint, float) eliminate allocation overhead  
- **Type Safety**: Full generic support with compile-time type checking
- **Proven Performance**: Comprehensive benchmarks against `sync.Map` and `sync.RWMutex` solutions
- **Configurable**: Tune shard count based on your concurrency patterns
- **Extensible**: Custom hash functions for specialized key types

## Installation

```bash
go get github.com/rugwirobaker/shmap
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/rugwirobaker/shmap"
)

func main() {
    // Create with sensible defaults
    m := shmap.New[string, int]()
    
    // Basic operations
    m.Set("hello", 42)
    m.Set("world", 99)
    
    if value, ok := m.Get("hello"); ok {
        fmt.Printf("hello = %d\n", value) // hello = 42
    }
    
    // Iterate over all key-value pairs
    m.Range(func(key string, value int) bool {
        fmt.Printf("%s: %d\n", key, value)
        return true // continue iteration
    })
    
    m.Delete("hello")
}
```

## API Reference

### Creation

```go
// Default configuration (recommended for most cases)
m := shmap.New[string, int]()

// Custom shard count (rounded up to next power of 2)
m := shmap.WithShards[string, int](64)

// Custom hash function for specialized key types
hasher := func(key MyKeyType) uint64 { /* custom logic */ }
m := shmap.WithHasher[MyKeyType, int](hasher)
```

### Operations

```go
// Set a key-value pair
m.Set(key, value)

// Get a value (returns value, found)
value, ok := m.Get(key)

// Delete a key
m.Delete(key)

// Iterate over all entries
m.Range(func(key K, value V) bool {
    // Process key-value pair
    return true // return false to stop iteration
})
```

## Performance Characteristics

### When to Use shmap

**Ideal for:**
- High-concurrency read/write workloads
- Multiple goroutines accessing shared data
- Applications with mixed read/write patterns
- Systems where lock contention is a bottleneck

**Consider alternatives for:**
- Single-threaded access patterns
- Extremely write-heavy workloads with few keys
- Memory-constrained environments where overhead matters

### Benchmark Results

Here's how `shmap` performs against standard alternatives on Apple M1 (darwin/arm64):

#### Core Performance (Mixed Read/Write Workloads)
```
# Read-Heavy (90% reads, 10% writes)
BenchmarkReadMostly/shmap-8       252730360    23.84 ns/op     0 B/op    0 allocs/op
BenchmarkReadMostly/rwmap-8        57044974    88.25 ns/op     0 B/op    0 allocs/op  
BenchmarkReadMostly/syncmap-8     162063607    38.93 ns/op     7 B/op    0 allocs/op

# Write-Heavy (50% reads, 50% writes)
BenchmarkWriteHeavy/shmap-8       211281535    28.44 ns/op     0 B/op    0 allocs/op
BenchmarkWriteHeavy/rwmap-8        54269379   101.2 ns/op      0 B/op    0 allocs/op
BenchmarkWriteHeavy/syncmap-8     100000000    66.96 ns/op    36 B/op    1 allocs/op
```

**Key Performance Wins:**
- **3.7x faster** than `sync.RWMutex` on read-heavy workloads  
- **3.6x faster** than `sync.RWMutex` on write-heavy workloads
- **1.6x faster** than `sync.Map` on read-heavy workloads
- **2.4x faster** than `sync.Map` on write-heavy workloads
- **Zero allocations** across all workloads vs `sync.Map`'s allocation overhead

#### Goroutine Scaling (Contention Reduction)
```
# Performance with increasing goroutine counts (20% writes, 80% reads)
                                    1 goroutine     32 goroutines   
BenchmarkGoroutineScaling/shmap      36.46 ns/op     25.19 ns/op    (↑ 45% faster)
BenchmarkGoroutineScaling/rwmap      31.22 ns/op     67.37 ns/op    (↓ 116% slower)  
BenchmarkGoroutineScaling/syncmap    61.60 ns/op     20.01 ns/op    (↑ 208% faster)
```

**Scaling Characteristics:**
- **shmap**: Consistent performance improvement with more goroutines
- **rwmap**: Severe performance degradation due to lock contention  
- **syncmap**: Excellent scaling but starts from a higher baseline

#### Shard Count Optimization
```
BenchmarkShardScaling/16shards-8      134707905    41.77 ns/op    0 B/op    0 allocs/op
BenchmarkShardScaling/128shards-8     249203017    25.65 ns/op    0 B/op    0 allocs/op
BenchmarkShardScaling/512shards-8     289211349    19.39 ns/op    0 B/op    0 allocs/op
```

Performance scales linearly with shard count - **2.2x faster** with optimal sharding.

#### Zero-Allocation Proof
```
# Hash functions (zero-allocation for common types)
BenchmarkHashFunctions/StringHasher-8  442413914   13.15 ns/op    0 B/op    0 allocs/op
BenchmarkHashFunctions/IntHasher-8     1000000000   0.32 ns/op    0 B/op    0 allocs/op

# Operational allocations (during normal usage)
BenchmarkOperationalAllocations/shmap_operations-8    258979004   22.39 ns/op   0 B/op   0 allocs/op
BenchmarkOperationalAllocations/syncmap_operations-8  155281969   38.72 ns/op   7 B/op   0 allocs/op
```

**Construction vs Operation Trade-off:**
```
# One-time construction cost
BenchmarkConstructionCost/shmap-8     1957310    3060 ns/op    11072 B/op   131 allocs/op
BenchmarkConstructionCost/rwmap-8     663443008     9 ns/op        0 B/op     0 allocs/op
```

Higher construction cost (due to 128 shards) but zero operational allocations.

#### Edge Cases
```
# Single-key contention (worst case for sharding)
BenchmarkHighContention/shmap-8      126843325    48.07 ns/op    0 B/op    0 allocs/op  
BenchmarkHighContention/syncmap-8    201900019    29.07 ns/op     7 B/op    0 allocs/op

# Range operations (6.9x faster than sync.Map)
BenchmarkRangeOperations/shmap-8         9061   649423 ns/op     0 B/op    0 allocs/op
BenchmarkRangeOperations/syncmap-8       1318  4500220 ns/op     0 B/op    0 allocs/op
```

**Honest Assessment:**
- **shmap wins**: Multi-goroutine workloads, mixed read/write patterns, zero allocations
- **sync.Map wins**: Single-key contention, very high concurrency scenarios  
- **Trade-off**: Higher construction cost for better operational performance

*Run `go test -bench=. -benchtime=5s -benchmem` to see results on your hardware.*

## Configuration Guide

### Choosing Shard Count

```go
// Default: GOMAXPROCS * 8, clamped to [64, 1024]
m := shmap.New[string, int]()

// Low contention scenarios
m := shmap.WithShards[string, int](16)

// High contention with many goroutines  
m := shmap.WithShards[string, int](256)

// Extreme concurrency
m := shmap.WithShards[string, int](512)
```

**Rule of thumb:** Start with defaults, then tune based on profiling. More shards reduce contention but increase memory overhead.

### Supported Key Types (Zero-Allocation)

Built-in optimized hashers for:
- `string`
- `int`, `int8`, `int16`, `int32`, `int64`  
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `uintptr`
- `float32`, `float64`

### Custom Key Types

```go
type UserID struct {
    Tenant string
    ID     uint64
}

func hashUserID(u UserID) uint64 {
    // Combine tenant string hash with ID
    h := shmap.StringHasher(u.Tenant)
    return shmap.Mix64(h ^ u.ID)
}

m := shmap.WithHasher[UserID, User](hashUserID)
```

## Architecture

`shmap` uses a simple but effective approach:

1. **Sharding**: The key space is divided into 2^N shards (power of 2 for fast modulo via bit masking)
2. **Hashing**: Each key is hashed to determine its shard: `shard = hash(key) & mask`
3. **Locking**: Each shard has its own `sync.RWMutex`, dramatically reducing contention
4. **Zero-alloc hashing**: Custom hashers avoid allocations for common key types

```
┌─────────────────────────────────────────────────────────┐
│                    shmap.Map[K,V]                       │
├─────────────────────────────────────────────────────────┤
│  hash(key) & mask → shard index                         │
├──────────────┬──────────────┬──────────────┬────────────┤
│   Shard 0    │   Shard 1    │   Shard 2    │    ...     │
│ ┌──────────┐ │ ┌──────────┐ │ ┌──────────┐ │            │
│ │RWMutex   │ │ │RWMutex   │ │ │RWMutex   │ │            │
│ │map[K]V   │ │ │map[K]V   │ │ │map[K]V   │ │            │
│ └──────────┘ │ └──────────┘ │ └──────────┘ │            │
└──────────────┴──────────────┴──────────────┴────────────┘
```

## Thread Safety

- **Concurrent reads**: Multiple goroutines can safely read simultaneously
- **Concurrent writes**: Writes to different shards don't block each other  
- **Mixed operations**: Reads and writes can happen concurrently
- **Range iteration**: Not linearizable - may see inconsistent snapshots during concurrent modifications

## Comparisons

| Feature | shmap | sync.Map | RWMutex + map |
|---------|-------|----------|---------------|
| Type Safety | ✅ Generics | ❌ interface{} | ✅ User-defined |
| Read Performance | ✅ Excellent | ✅ Good | ⚠️ Good (RLock) |  
| Write Performance | ✅ Excellent | ⚠️ Moderate | ❌ Poor (Lock) |
| Memory Overhead | ⚠️ Moderate | ✅ Low | ✅ Low |
| Predictable Performance | ✅ Yes | ❌ GC-dependent | ✅ Yes |

## Contributing

We welcome contributions! Please see our [contributing guidelines](CONTRIBUTING.md) for details.

### Running Tests

```bash
# Basic tests
go test ./...

# Race detection  
go test -race ./...

# Benchmarks
go test -bench=. -benchmem

# Performance comparison
go test -bench=BenchmarkShmap -bench=BenchmarkRWMap -bench=BenchmarkSyncMap
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
