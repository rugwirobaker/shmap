package shmap

import (
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"testing"
)

func TestSetGetDelete(t *testing.T) {
	m := New[string, int]() // defaults
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("b", 3)

	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Fatalf("get a: want 1, ok=true; got %v, %v", v, ok)
	}
	if v, ok := m.Get("b"); !ok || v != 3 {
		t.Fatalf("get b: want 3, ok=true; got %v, %v", v, ok)
	}
	if _, ok := m.Get("c"); ok {
		t.Fatalf("get c: want ok=false")
	}

	m.Delete("a")
	if _, ok := m.Get("a"); ok {
		t.Fatalf("post-delete get a: want ok=false")
	}
}

func TestRange(t *testing.T) {
	m := WithShards[string, int](16)
	want := map[string]int{"k1": 1, "k2": 2, "k3": 3}
	for k, v := range want {
		m.Set(k, v)
	}

	got := make(map[string]int)
	m.Range(func(k string, v int) bool { got[k] = v; return true })

	if len(got) != len(want) {
		t.Fatalf("range size mismatch: got %d want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("range mismatch for %s: got %d want %d", k, got[k], v)
		}
	}
}

func TestRangeEarlyStop(t *testing.T) {
	m := WithShards[string, int](16)
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	count := 0
	m.Range(func(k string, v int) bool { count++; return count < 10 })
	if count != 10 {
		t.Fatalf("expected early stop at 10, got %d", count)
	}
}

func TestConcurrentSetGet(t *testing.T) {
	m := WithShards[string, int](32)
	n := 1000
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = "k" + strconv.Itoa(i)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() { // writer
		defer wg.Done()
		r := rand.New(rand.NewSource(42))
		for i := 0; i < 20000; i++ {
			k := keys[r.Intn(n)]
			m.Set(k, i)
		}
	}()
	go func() { // reader
		defer wg.Done()
		r := rand.New(rand.NewSource(43))
		for i := 0; i < 20000; i++ {
			k := keys[r.Intn(n)]
			_, _ = m.Get(k)
		}
	}()
	go func() { // ranger
		defer wg.Done()
		for i := 0; i < 20000; i++ {
			m.Range(func(k string, v int) bool { return true })
		}
	}()

	wg.Wait()

	var gotKeys []string
	m.Range(func(k string, v int) bool { gotKeys = append(gotKeys, k); return true })
	if len(gotKeys) == 0 {
		t.Fatalf("expected some keys after concurrent ops")
	}
	sorted := append([]string(nil), gotKeys...)
	sort.Strings(sorted)
	if len(sorted) != len(gotKeys) {
		t.Fatalf("copy/sort mismatch")
	}
}
