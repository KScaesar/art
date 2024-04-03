package Artifex

import (
	"math/rand"
	"testing"
)

func Benchmark_Hub_Join(b *testing.B) {
	b.ReportAllocs()

	b.StopTimer()
	type Obj struct{}
	obj := &Obj{}

	stopObj := func(t *Obj) { return }
	hub := NewHub(stopObj)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		hub.Join("benchmark", obj)
	}

}

func Benchmark_Hub_UpdateByOldKey(b *testing.B) {
	b.ReportAllocs()

	b.StopTimer()
	type Obj struct{}

	stopObj := func(t *Obj) { return }
	hub := NewHub(stopObj)

	var keys []string
	size := 300000
	for i := 0; i < size; i++ {
		obj := &Obj{}
		key := GenerateRandomCode(8)
		keys = append(keys, key)
		hub.Join(key, obj)
	}

	freshKey := GenerateRandomCode(8)
	update := func(obj *Obj) (string, error) {
		return freshKey, nil
	}

	b.StartTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			oldKey := keys[rand.Intn(size)]
			hub.UpdateByOldKey(oldKey, update)
		}
	})

	// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
	// UpdateByOldKey
	// Benchmark_Hub_UpdateByOldKey-12    	 1638529	       689.2 ns/op	     135 B/op	       4 allocs/op
	// UpdateByOldKey2
	// Benchmark_Hub_UpdateByOldKey-12    	 4955065	       264.9 ns/op	     144 B/op	       5 allocs/op
}
