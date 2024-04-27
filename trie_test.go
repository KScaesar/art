package Artifex

import (
	"testing"
)

func Test_trie_endpoint(t *testing.T) {
	node := newTrie("/")

	mw := &paramHandler{middlewares: []Middleware{endpoint_mw()}}
	node.addRoute("", 0, mw, []Middleware{})

	defaultHandler := &paramHandler{defaultHandler: endpoint_default}
	node.addRoute("topic", 0, defaultHandler, []Middleware{})

	handler1 := &paramHandler{handler: endpoint1}
	handler2 := &paramHandler{handler: endpoint2}
	node.addRoute("topic1/", 0, handler1, []Middleware{})
	node.addRoute("topic2/users/", 0, handler2, []Middleware{})
	node.addRoute("topic2/orders/", 0, handler2, []Middleware{})
	node.addRoute("{topic}/game2/kindA/", 0, handler1, []Middleware{})

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

func endpoint_default(_ *Message, _ any) error {
	return nil
}

func endpoint1(_ *Message, _ any) error {
	return nil
}

func endpoint2(_ *Message, _ any) error {
	return nil
}

func endpoint_mw() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(_ *Message, _ any) error {
			return nil
		}
	}
}
