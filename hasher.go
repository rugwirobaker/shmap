package shmap

import "math"

// Mix64 is a fast 64-bit mixing function (SplitMix64 finalizer).
func Mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

// StringHasher — FNV-1a (zero-alloc)
func StringHasher(s string) uint64 {
	var h uint64 = 1469598103934665603
	const p = 1099511628211
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= p
	}
	return h
}

// Integer & float helpers (zero-alloc)
type IntLike interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}
type UintLike interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}
type FloatLike interface{ ~float32 | ~float64 }

func IntHasher[T IntLike](v T) uint64   { return Mix64(uint64(int64(v))) }
func UintHasher[T UintLike](v T) uint64 { return Mix64(uint64(v)) }
func FloatHasher[T FloatLike](v T) uint64 {
	switch any(v).(type) {
	case float32:
		return Mix64(uint64(math.Float32bits(any(v).(float32))))
	default:
		return Mix64(math.Float64bits(any(v).(float64)))
	}
}

// DefaultHasher installs a zero-alloc hasher for common K.
// Supported: string, all int/uint widths, float32/64.
// For other key types (e.g., structs), call WithHasher.
func DefaultHasher[K comparable]() (HashFn[K], bool) {
	var zero K
	switch any(zero).(type) {
	case string:
		return func(k K) uint64 { return StringHasher(any(k).(string)) }, true
	case int:
		return func(k K) uint64 { return IntHasher(any(k).(int)) }, true
	case int8:
		return func(k K) uint64 { return IntHasher(any(k).(int8)) }, true
	case int16:
		return func(k K) uint64 { return IntHasher(any(k).(int16)) }, true
	case int32:
		return func(k K) uint64 { return IntHasher(any(k).(int32)) }, true
	case int64:
		return func(k K) uint64 { return IntHasher(any(k).(int64)) }, true
	case uint:
		return func(k K) uint64 { return UintHasher(any(k).(uint)) }, true
	case uint8:
		return func(k K) uint64 { return UintHasher(any(k).(uint8)) }, true
	case uint16:
		return func(k K) uint64 { return UintHasher(any(k).(uint16)) }, true
	case uint32:
		return func(k K) uint64 { return UintHasher(any(k).(uint32)) }, true
	case uint64:
		return func(k K) uint64 { return UintHasher(any(k).(uint64)) }, true
	case uintptr:
		return func(k K) uint64 { return UintHasher(any(k).(uintptr)) }, true
	case float32:
		return func(k K) uint64 { return FloatHasher(any(k).(float32)) }, true
	case float64:
		return func(k K) uint64 { return FloatHasher(any(k).(float64)) }, true
	default:
		return nil, false
	}
}
