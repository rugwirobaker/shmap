package shmap

import "sync"

// HashFn hashes a key K to a 64-bit value. Must be deterministic.
type HashFn[K comparable] func(K) uint64

// Map is a lock-sharded hash map.
// - Concurrency: one RWMutex per shard
// - Indexing: shard = hash(key) & mask
type Map[K comparable, V any] struct {
	shards []shard[K, V]
	mask   uint64
	hash   HashFn[K]
}

// New creates a sharded map with sane defaults:
// - shard count = DefaultShards() (rounded to power-of-two)
// - hasher      = DefaultHasher[K]() for common K (string/int/uint/float)
func New[K comparable, V any]() *Map[K, V] {
	return build[K, V](DefaultShards(), nil)
}

// WithShards creates a map using your shard count (rounded up to power-of-two).
// Example: WithShards(100) -> 128 shards.
func WithShards[K comparable, V any](shards int) *Map[K, V] {
	if shards < 1 {
		shards = 1
	}
	return build[K, V](shards, nil)
}

// WithHasher creates a map using a custom hash function and default shard count.
func WithHasher[K comparable, V any](h HashFn[K]) *Map[K, V] {
	if h == nil {
		panic("shmap: WithHasher requires a non-nil HashFn")
	}
	m := build[K, V](DefaultShards(), h)
	return m
}

func build[K comparable, V any](shards int, h HashFn[K]) *Map[K, V] {
	n := 1 << bitsFor(shards) // power-of-two shards
	ss := make([]shard[K, V], n)
	for i := range ss {
		ss[i].m = make(map[K]V)
	}
	if h == nil {
		dh, ok := DefaultHasher[K]()
		if !ok {
			panic("shmap: no default hasher for this key type; use WithHasher")
		}
		h = dh
	}
	return &Map[K, V]{shards: ss, mask: uint64(n - 1), hash: h}
}

type shard[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func (m *Map[K, V]) idx(k K) int {
	return int(m.hash(k) & m.mask)
}

// === Map operations ===

// Get returns v,ok for key k.
func (m *Map[K, V]) Get(k K) (V, bool) {
	i := m.idx(k)
	m.shards[i].mu.RLock()
	v, ok := m.shards[i].m[k]
	m.shards[i].mu.RUnlock()
	return v, ok
}

// Set sets key k to value v.
func (m *Map[K, V]) Set(k K, v V) {
	i := m.idx(k)
	m.shards[i].mu.Lock()
	m.shards[i].m[k] = v
	m.shards[i].mu.Unlock()
}

// Delete removes key k if present.
func (m *Map[K, V]) Delete(k K) {
	i := m.idx(k)
	m.shards[i].mu.Lock()
	delete(m.shards[i].m, k)
	m.shards[i].mu.Unlock()
}

// Range iterates until fn returns false.
// Order is undefined and not linearizable.
func (m *Map[K, V]) Range(fn func(K, V) bool) {
	for i := range m.shards {
		s := &m.shards[i]
		s.mu.RLock()
		for k, v := range s.m {
			if !fn(k, v) {
				s.mu.RUnlock()
				return
			}
		}
		s.mu.RUnlock()
	}
}
