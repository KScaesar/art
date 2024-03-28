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
// handle message 執行路徑, 依照欄位的順序, 從上到下
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
			panic(subject + ": assign duplicated getSubject")
		}
		node.routeHandler.getSubject = route.getSubject
	}
	if route.middlewares != nil {
		node.routeHandler.middlewares = append(node.routeHandler.middlewares, route.middlewares...)
	}
	if route.handler != nil {
		if node.routeHandler.handler != nil {
			panic(subject + ": assign duplicated handler")
		}
		node.routeHandler.handler = route.handler
	}
	if route.defaultHandler != nil {
		if node.routeHandler.defaultHandler != nil {
			panic(subject + ": assign duplicated defaultHandler")
		}
		node.routeHandler.defaultHandler = route.defaultHandler
	}
	if route.notFoundHandler != nil {
		if node.routeHandler.notFoundHandler != nil {
			panic(subject + ": assign duplicated notFoundHandler")
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

func newTrie[M any](delimiter string) *trie[M] {
	return &trie[M]{
		staticChild: make(map[byte]*trie[M]),
		delimiter:   delimiter,
	}
}

type trie[M any] struct {
	staticChild       map[byte]*trie[M]
	wildcardChildWord string
	wildcardChild     *trie[M]

	delimiter   string
	fullSubject string
	routeHandler[M]
}

func (node *trie[M]) addRoute(subject string, cursor int, handler *routeHandler[M]) *trie[M] {
	if len(subject) == cursor {
		handler.register(subject, node)
		return node
	}

	char := subject[cursor]
	if char != '{' {
		child, exist := node.staticChild[char]
		if !exist {
			child = newTrie[M](node.delimiter)
			child.fullSubject = node.fullSubject + string(char)
			node.staticChild[char] = child
		}
		return child.addRoute(subject, cursor+1, handler)
	}

	if node.delimiter == "" {
		panic(subject + ": route delimiter is empty: not support wildcard")
	}

	idx := cursor
	for idx < len(subject) && subject[idx] != '}' {
		idx++
	}

	if subject[idx] != '}' {
		panic(subject + ": lack wildcard '}'")
	}

	if node.wildcardChild != nil {
		panic(subject + ": assign duplicated wildcard")
	}

	child := newTrie[M](node.delimiter)
	child.fullSubject = node.fullSubject + subject[cursor:idx+1] // {word}, include {}
	node.wildcardChildWord = subject[cursor+1 : idx]             // word
	node.wildcardChild = child
	return child.addRoute(subject, idx+1, handler)
}

func (node *trie[M]) handleMessage(subject string, cursor int, path *routeHandler[M], msg *M, route *RouteParam) (err error) {
	idx := cursor
	current := node

	var wildcardStart int
	var wildcardParent *trie[M]

	path.collect(current)
	if current.transforms != nil {
		idx = 0
		err = current.transform(msg, route)
		if err != nil {
			return err
		}
		subject, err = path.getSubject(msg)
		if err != nil {
			return err
		}
	}

	for idx < len(subject) {
		if current.wildcardChild != nil {
			wildcardStart = idx
			wildcardParent = current
		}

		child, exist := current.staticChild[subject[idx]]
		if !exist {
			break
		}
		idx++
		current = child

		path.collect(current)
		if current.transforms != nil {
			idx = 0
			err = current.transform(msg, route)
			if err != nil {
				return err
			}
			subject, err = path.getSubject(msg)
			if err != nil {
				return err
			}
		}
	}

	// for static route
	if idx == len(subject) {
		if current.handler != nil {
			return LinkMiddlewares(current.handler, path.middlewares...)(msg, route)
		}
		if path.defaultHandler != nil {
			return LinkMiddlewares(path.defaultHandler, path.middlewares...)(msg, route)
		}
		if path.notFoundHandler != nil {
			return path.notFoundHandler(msg, route)
		}
		if wildcardParent == nil {
			return ErrorWrapWithMessage(ErrNotFoundSubjectOfMux, "mux subject")
		}
	}

	// for wildcard route
	wildcardFinish := wildcardStart
	for wildcardFinish < len(subject) && subject[wildcardFinish] != current.delimiter[0] {
		wildcardFinish++
	}

	route.Set(wildcardParent.wildcardChildWord, subject[wildcardStart:wildcardFinish])
	return wildcardParent.wildcardChild.handleMessage(subject, wildcardFinish, path, msg, route)
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
	for _, next := range node.staticChild {
		next._subjects_(r)
	}

	if node.wildcardChild == nil {
		return
	}
	node.wildcardChild._subjects_(r)
}
