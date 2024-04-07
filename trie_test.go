package Artifex

import (
	"testing"
)

func Test_trie_endpoint(t *testing.T) {
	node := newTrie[string]("/")

	mw := &paramHandler[string]{middlewares: []Middleware[string]{endpoint_mw()}}
	node.addRoute("", 0, mw, &pathHandler[string]{})

	defaultHandler := &paramHandler[string]{defaultHandler: endpoint_default}
	node.addRoute("topic", 0, defaultHandler, &pathHandler[string]{})

	handler1 := &paramHandler[string]{handler: endpoint1}
	handler2 := &paramHandler[string]{handler: endpoint2}
	node.addRoute("topic1/", 0, handler1, &pathHandler[string]{})
	node.addRoute("topic2/users/", 0, handler2, &pathHandler[string]{})
	node.addRoute("topic2/orders/", 0, handler2, &pathHandler[string]{})
	node.addRoute("{topic}/game2/kindA/", 0, handler1, &pathHandler[string]{})

	expectedSubjects := []string{
		"topic.*",
		"topic1/",
		"topic2/orders/",
		"topic2/users/",
		"{topic}/game2/kindA/",
	}

	endpoints := node.endpoint()
	for i, endpoint := range endpoints {
		if endpoint[0] != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", endpoint[0], expectedSubjects[i])
		}
		t.Logf("subject=%v fn=%v\n", endpoint[0], endpoint[1])
	}
}

func endpoint_default(_ *string, _ *RouteParam) error {
	return nil
}

func endpoint1(_ *string, _ *RouteParam) error {
	return nil
}

func endpoint2(_ *string, _ *RouteParam) error {
	return nil
}

func endpoint_mw() Middleware[string] {
	return func(next HandleFunc[string]) HandleFunc[string] {
		return func(_ *string, _ *RouteParam) error {
			return nil
		}
	}
}
