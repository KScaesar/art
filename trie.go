package Artifex

import (
	"errors"
	"fmt"
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
// handle message 執行順序, 依照號碼 0~5
type paramHandler[M any] struct {
	// 0
	transform HandleFunc[M]

	// 1
	getSubject NewSubjectFunc[M]

	// 2
	middlewares []Middleware[M]

	// 3
	handler     HandleFunc[M]
	handlerName string

	// 4
	defaultHandler     HandleFunc[M]
	defaultHandlerName string

	// 5
	notFoundHandler HandleFunc[M]
}

func (param *paramHandler[M]) register(leafNode *trie[M], path *pathHandler[M]) error {
	if param.transform != nil {
		if leafNode.transform != nil {
			return errors.New("assign duplicated transform")
		}
		leafNode.transform = param.transform
		leafNode.getSubject = path.getSubject
	}

	if param.getSubject != nil {
		leafNode.getSubject = param.getSubject
	}

	if param.middlewares != nil {
		leafNode.middlewares = append(leafNode.middlewares, param.middlewares...)
	}

	if param.handler != nil {
		if leafNode.handler != nil {
			return errors.New("assign duplicated handler")
		}
		leafNode.handler = LinkMiddlewares(param.handler, path.middlewares...)

		if param.handlerName == "" {
			leafNode.handlerName = functionName(param.handler)
		} else {
			leafNode.handlerName = param.handlerName
		}
	}

	if param.defaultHandler != nil {
		if leafNode.defaultHandler != nil {
			return errors.New("assign duplicated defaultHandler")
		}
		leafNode.defaultHandler = LinkMiddlewares(param.defaultHandler, path.middlewares...)

		if param.defaultHandlerName == "" {
			leafNode.defaultHandlerName = functionName(param.defaultHandler)
		} else {
			leafNode.defaultHandlerName = param.defaultHandlerName
		}
	}

	if param.notFoundHandler != nil {
		if leafNode.notFoundHandler != nil {
			return errors.New("assign duplicated notFoundHandler")
		}
		leafNode.notFoundHandler = param.notFoundHandler
	}

	return nil
}

func functionName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

type pathHandler[M any] struct {
	getSubject  NewSubjectFunc[M]
	middlewares []Middleware[M]
}

func (path *pathHandler[M]) collect(pathNode *trie[M]) {
	if pathNode.getSubject != nil {
		path.getSubject = pathNode.getSubject
	}
	if pathNode.middlewares != nil {
		path.middlewares = append(path.middlewares, pathNode.middlewares...)
	}
}

func newTrie[M any](delimiter string) *trie[M] {
	return &trie[M]{
		staticChild: make(map[byte]*trie[M]),
		delimiter:   delimiter,
	}
}

type trie[M any] struct {
	staticChild map[byte]*trie[M] // key:value = char:child

	wildcardChildWord string
	wildcardChild     *trie[M]

	delimiter   string
	fullSubject string
	paramHandler[M]
}

func (node *trie[M]) addRoute(subject string, cursor int, param *paramHandler[M], path *pathHandler[M]) *trie[M] {
	path.collect(node)

	if len(subject) == cursor {
		if param == nil { // for Mux.Group
			param = &paramHandler[M]{
				getSubject:  path.getSubject,
				middlewares: path.middlewares,
			}
		}

		leafNode := node
		err := param.register(leafNode, path)
		if err != nil {
			Err := fmt.Errorf("subject=%v: %w", subject, err)
			panic(Err)
		}
		return leafNode
	}

	char := subject[cursor]
	if char != '{' {
		child, exist := node.staticChild[char]
		if !exist {
			child = newTrie[M](node.delimiter)
			child.fullSubject = node.fullSubject + string(char)
			node.staticChild[char] = child
		}
		return child.addRoute(subject, cursor+1, param, path)
	}

	if node.delimiter == "" {
		err := fmt.Errorf("subject=%v: route delimiter is empty: not support wildcard", subject)
		panic(err)
	}

	idx := cursor
	for idx < len(subject) && subject[idx] != '}' {
		idx++
	}

	if subject[idx] != '}' {
		err := fmt.Errorf("subject=%v: lack wildcard '}'", subject)
		panic(err)
	}

	if node.wildcardChild != nil {
		err := fmt.Errorf("subject=%v: assign duplicated wildcard: %v", subject, node.fullSubject)
		panic(err)
	}

	child := newTrie[M](node.delimiter)
	child.fullSubject = node.fullSubject + subject[cursor:idx+1] // {word}, include {}
	node.wildcardChildWord = subject[cursor+1 : idx]             // word, exclude {}
	node.wildcardChild = child
	return child.addRoute(subject, idx+1, param, path)
}

func (node *trie[M]) handleMessage(subject string, cursor int, message *M, dep any, route *RouteParam) (err error) {
	current := node

	var defaultHandler, notFoundHandler HandleFunc[M]
	var wildcardStart int
	var wildcardParent *trie[M]

	for cursor <= len(subject) {
		if current.transform != nil {
			err = current.transform(message, dep, route)
			if err != nil {
				return err
			}
			subject = current.getSubject(message)
		}

		if current.defaultHandler != nil {
			defaultHandler = current.defaultHandler
		}

		if current.notFoundHandler != nil {
			notFoundHandler = current.notFoundHandler
		}

		if current.wildcardChild != nil {
			wildcardStart = cursor
			wildcardParent = current
		}

		if cursor == len(subject) {
			break
		}

		child, exist := current.staticChild[subject[cursor]]
		if !exist {
			break
		}
		cursor++
		current = child
	}

	// for static route
	if current.handler != nil {
		return current.handler(message, dep, route)
	}
	if wildcardParent == nil {
		if defaultHandler != nil {
			return defaultHandler(message, dep, route)
		}
		if notFoundHandler != nil {
			return notFoundHandler(message, dep, route)
		}
		return ErrNotFoundSubject
	}

	// for wildcard route
	wildcardFinish := wildcardStart
	for wildcardFinish < len(subject) && subject[wildcardFinish] != current.delimiter[0] {
		wildcardFinish++
	}

	route.Set(wildcardParent.wildcardChildWord, subject[wildcardStart:wildcardFinish])

	err = wildcardParent.wildcardChild.handleMessage(subject, wildcardFinish, message, dep, route)
	if err != nil && errors.Is(err, ErrNotFoundSubject) {
		if defaultHandler != nil {
			return defaultHandler(message, dep, route)
		}
		if notFoundHandler != nil {
			return notFoundHandler(message, dep, route)
		}
		return ErrNotFoundSubject
	}
	return err
}

// pair = [subject, function]
func (node *trie[M]) endpoint() (pairs [][2]string) {
	pairs = make([][2]string, 0)
	node._endpoint_(&pairs)

	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i][0] < pairs[j][0]
	})
	return
}

func (node *trie[M]) _endpoint_(paris *[][2]string) {
	if node.handler != nil {
		*paris = append(*paris, [2]string{node.fullSubject, node.handlerName})
	}
	if node.defaultHandler != nil {
		*paris = append(*paris, [2]string{node.fullSubject + ".*", node.defaultHandlerName})
	}

	for _, next := range node.staticChild {
		next._endpoint_(paris)
	}

	if node.wildcardChild == nil {
		return
	}
	node.wildcardChild._endpoint_(paris)
}
