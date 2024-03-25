package Artifex

import (
	"reflect"
	"runtime"
	"sort"
)

// getSubject 是為了避免 generic type 呼叫 method 所造成的效能降低
// 也可以因應不同情境, 改變取得 subject 的規則
//
// https://www.youtube.com/watch?v=D1hI55EcBB4&t=20260s
//
// https://hackmd.io/@fieliapm/BkHvJjYq3#/5/2
//
// hande message 執行路徑, 依照欄位的順序, 從上到下
type routeHandler[M any] struct {
	transforms  []HandleFunc[M]
	getSubject  NewSubjectFunc[M]
	middlewares []Middleware[M]
	handler     HandleFunc[M]

	defaultHandler  HandleFunc[M]
	notFoundHandler HandleFunc[M]
}

func (path *routeHandler[M]) collect(node *trie[M]) {
	if node.middlewares != nil {
		path.middlewares = append(path.middlewares, node.middlewares...)
	}
	if node.getSubject != nil {
		path.getSubject = node.getSubject
	}
	if node.defaultHandler != nil {
		path.defaultHandler = node.defaultHandler
	}
	if node.notFoundHandler != nil {
		path.notFoundHandler = node.notFoundHandler
	}
}

func (route *routeHandler[M]) register(subject string, node *trie[M]) {
	if route.transforms != nil {
		node.routeHandler.transforms = append(node.routeHandler.transforms, route.transforms...)
	}
	if route.getSubject != nil {
		if node.routeHandler.getSubject != nil {
			panic("assign duplicated getSubject: " + subject)
		}
		node.routeHandler.getSubject = route.getSubject
	}
	if route.middlewares != nil {
		node.routeHandler.middlewares = append(node.routeHandler.middlewares, route.middlewares...)
	}
	if route.handler != nil {
		if node.routeHandler.handler != nil {
			panic("assign duplicated handler: " + subject)
		}
		node.routeHandler.handler = route.handler
	}
	if route.defaultHandler != nil {
		if node.routeHandler.defaultHandler != nil {
			panic("assign duplicated defaultHandler: " + subject)
		}
		node.routeHandler.defaultHandler = route.defaultHandler
	}
	if route.notFoundHandler != nil {
		if node.routeHandler.notFoundHandler != nil {
			panic("assign duplicated notFoundHandler: " + subject)
		}
		node.routeHandler.notFoundHandler = route.notFoundHandler
	}
}

func (handler *routeHandler[M]) reset() {
	handler.transforms = handler.transforms[:0]
	handler.getSubject = nil
	handler.middlewares = handler.middlewares[:0]
	handler.handler = nil
	handler.defaultHandler = nil
	handler.notFoundHandler = nil
}

type trie[M any] struct {
	child       map[string]*trie[M]
	part        string
	fullSubject string
	routeHandler[M]
}

func newTrie[M any]() *trie[M] {
	return &trie[M]{
		child: make(map[string]*trie[M]),
	}
}

func (node *trie[M]) addRoute(subject string, cursor int, handler *routeHandler[M]) *trie[M] {
	if len(subject) == cursor {
		handler.register(subject, node)
		return node
	}

	part := string(subject[cursor])
	next, exist := node.child[part]
	if !exist {
		next = newTrie[M]()
		next.part = part
		next.fullSubject = node.fullSubject + part
		node.child[part] = next
	}

	return next.addRoute(subject, cursor+1, handler)
}

func (node *trie[M]) handleMessage(subject string, cursor int, path *routeHandler[M], msg *M, route *RouteParam) (err error) {
Loop:
	path.collect(node)

	if node.transforms != nil {
		cursor = 0
		err = node.transform(msg, route)
		if err != nil {
			return err
		}

		subject, err = path.getSubject(msg)
		if err != nil {
			return err
		}
	}

	if len(subject) == cursor {
		if node.handler != nil {
			return LinkMiddlewares(node.handler, path.middlewares...)(msg, route)
		}
		if path.defaultHandler != nil {
			return LinkMiddlewares(path.defaultHandler, path.middlewares...)(msg, route)
		}
		if path.notFoundHandler != nil {
			return path.notFoundHandler(msg, route)
		}
		return ErrorWrapWithMessage(ErrNotFound, "mux subject")
	}

	word := string(subject[cursor])
	next, exist := node.child[word]
	if exist {
		node = next
		cursor++
		goto Loop
	}

	if path.defaultHandler != nil {
		return LinkMiddlewares(path.defaultHandler, path.middlewares...)(msg, route)
	}

	if path.notFoundHandler != nil {
		return path.notFoundHandler(msg, route)
	}

	return ErrorWrapWithMessage(ErrNotFound, "mux subject")
}

func (node *trie[M]) transform(message *M, route *RouteParam) error {
	for _, transform := range node.transforms {
		err := transform(message, route)
		if err != nil {
			return err
		}
	}
	return nil
}

func (node *trie[M]) endpoint() (subjects, functions []string) {
	result := make(map[string]string)
	node._subjects_(result)

	for s := range result {
		subjects = append(subjects, s)
	}
	sort.SliceStable(subjects, func(i, j int) bool {
		return subjects[i] < subjects[j]
	})
	for _, s := range subjects {
		functions = append(functions, result[s])
	}
	return
}

func functionName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func (node *trie[M]) _subjects_(r map[string]string) {
	if node.handler != nil {
		r[node.fullSubject] = functionName(node.handler)
	}
	for _, next := range node.child {
		next._subjects_(r)
	}
}
