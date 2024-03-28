package Artifex

import (
	"testing"
)

func Test_trie_endpoint(t *testing.T) {
	node := newTrie[string]("")
	handler1 := func(_ *string, _ *RouteParam) error {
		return nil
	}
	handler2 := func(_ *string, _ *RouteParam) error {
		return nil
	}
	param1 := &routeHandler[string]{
		transforms:     nil,
		getSubject:     nil,
		defaultHandler: nil,
		middlewares:    nil,
		handler:        handler1,
	}
	param2 := &routeHandler[string]{
		transforms:     nil,
		getSubject:     nil,
		defaultHandler: nil,
		middlewares:    nil,
		handler:        handler2,
	}

	node.addRoute("topic1/", 0, param1)
	node.addRoute("topic2/users/", 0, param2)
	node.addRoute("topic2/orders/", 0, param2)
	node.addRoute("topic4/game2/kindA/", 0, param1)

	expectedSubjects := []string{
		"topic1/",
		"topic2/orders/",
		"topic2/users/",
		"topic4/game2/kindA/",
	}

	gotSubjects, gotFunctions := node.endpoint()
	for i, gotSubject := range gotSubjects {
		if gotSubject != expectedSubjects[i] {
			t.Errorf("unexpected output: got %s, want %s", gotSubject, expectedSubjects[i])
		}
	}
	_ = gotFunctions
}
