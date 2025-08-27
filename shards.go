package shmap

import (
	"math/bits"
	"runtime"
)

// DefaultShards picks a reasonable shard count for this process.
// Heuristic: round_up_pow2(GOMAXPROCS * 8), clamped to [64, 1024].
func DefaultShards() int {
	p := runtime.GOMAXPROCS(0)
	target := p * 8
	if target < 64 {
		target = 64
	}
	if target > 1024 {
		target = 1024
	}
	// round up to next power-of-two
	if target&(target-1) == 0 { // already power-of-two
		return target
	}
	return 1 << bits.Len(uint(target))
}

// bitsFor returns log2(rounded_up_pow2(shards)).
func bitsFor(shards int) int {
	if shards <= 1 {
		return 0
	}
	if shards&(shards-1) == 0 {
		return bits.Len(uint(shards - 1))
	}
	return bits.Len(uint(shards))
}
