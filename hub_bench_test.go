package art

import (
	"math/rand"
	"testing"
)

func Benchmark_Hub_Join(b *testing.B) {
	b.ReportAllocs()

	b.StopTimer()

	adp := &Websocket{}

	hub := NewHub()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		hub.Join("benchmark", adp)
	}

	// cpu: 13th Gen Intel(R) Core(TM) i5-1340P
	// Benchmark_Hub_Join
	// Benchmark_Hub_Join-16    	 8515387	       125.2 ns/op	      48 B/op	       3 allocs/op
}

type Websocket struct {
	IAdapter
}

func (ws *Websocket) Stop() error {
	return nil
}

func Benchmark_Hub_UpdateByOldKey(b *testing.B) {
	b.ReportAllocs()

	b.StopTimer()

	hub := NewHub()

	var keys []string
	size := 300000
	for i := 0; i < size; i++ {
		adp := &Redis{}
		key := GenerateRandomCode(8)
		keys = append(keys, key)
		hub.Join(key, adp)
	}

	freshKey := GenerateRandomCode(8)
	update := func(adp IAdapter) string {
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

type Redis struct {
	name string
	IAdapter
}

func (r *Redis) Stop() error {
	return nil
}
