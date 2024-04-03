package Artifex

import (
	"testing"
)

func Benchmark_Hub_Join(b *testing.B) {
	b.ReportAllocs()

	b.StopTimer()
	type Obj struct{}
	obj := &Obj{}

	stopObj := func(t *Obj) error { return nil }
	hub := NewHub(stopObj)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		hub.Join("benchmark", obj)
	}

}
