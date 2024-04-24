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
	type Obj struct {
		name string
	}

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
	update := func(obj *Obj) string {
		return freshKey
	}

	b.StartTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			oldKey := keys[rand.Intn(size)]
			hub.UpdateByOldKey(oldKey, update)
		}
	})

	// cpu: 13th Gen Intel(R) Core(TM) i5-1340P
	// Benchmark_Hub_UpdateByOldKey
	// Benchmark_Hub_UpdateByOldKey-16    	 1000000	      1385 ns/op	     154 B/op	       4 allocs/op
}
