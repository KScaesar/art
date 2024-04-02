package Artifex

import (
	"testing"
)

func Benchmark_Hub_Join(b *testing.B) {
	b.ReportAllocs()

	opt := NewPublisherOption[string]().
		AdapterSend(func(_ IAdapter, _ *string) error {
			return nil
		}).
		AdapterStop(func(_ IAdapter, _ *string) error {
			return nil
		}).
		Identifier("benchmark")

	hub := NewHub(func(t *Adapter[struct{}, string]) error { return t.Stop() })

	for i := 0; i < b.N; i++ {
		publisher, err := NewPublisher(opt)
		if err != nil {
			panic(err)
		}
		hub.Join(publisher.Identifier(), publisher)
	}

	// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
	// Benchmark_Hub_Join-12    	 1503979	       794.9 ns/op	     464 B/op	       7 allocs/op
	// Benchmark_Hub_Join-12    	 1515572	       803.8 ns/op	     464 B/op	       7 allocs/op
	// Benchmark_Hub_Join-12    	 1449607	       817.5 ns/op	     464 B/op	       7 allocs/op
}
