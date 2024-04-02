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
			// fmt.Println("stop")
			return nil
		}).
		Identifier("benchmark")

	hub := NewHub(func(t *Adapter[struct{}, string]) error { return t.Stop() })
	publisher, _ := NewPublisher(opt)

	for i := 0; i < b.N; i++ {
		hub.Join2(publisher.Identifier(), publisher)
	}

}
