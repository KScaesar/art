package Artifex

import (
	"reflect"
	"runtime"
	"sort"
)

type nodeKind uint8

const (
	normalKind nodeKind = iota
	rootKind
)

type handlerParam[M any] struct {
	// getSubject 是為了避免 generic type 呼叫 method 所造成的效能降低
	// 也可以因應不同情境, 改變取得 subject 的規則
	//
	// https://www.youtube.com/watch?v=D1hI55EcBB4&t=20260s
	//
	// https://hackmd.io/@fieliapm/BkHvJjYq3#/5/2
	getSubject      NewSubjectFunc[M]
	transforms      []func(old M) (fresh M, err error)
	defaultHandler  MessageHandler[M]
	middlewares     []MessageDecorator[M]
	handler         MessageHandler[M]
	notFoundHandler MessageHandler[M]
}

type trie[M any] struct {
	kind        nodeKind
	child       map[string]*trie[M]
	fullSubject string
	handlerParam[M]
}

func (node *trie[M]) insert(subject string, idx int, param handlerParam[M]) *trie[M] {
	if len(subject) == idx {
		if param.transforms != nil {
			node.handlerParam.transforms = append(node.handlerParam.transforms, param.transforms...)
		}
		if param.getSubject != nil {
			node.handlerParam.getSubject = param.getSubject
		}
		if param.defaultHandler != nil {
			if node.handlerParam.defaultHandler != nil {
				panic("assign duplicated defaultHandler: " + subject)
			}
			node.handlerParam.defaultHandler = param.defaultHandler
		}
		if param.middlewares != nil {
			node.handlerParam.middlewares = append(node.handlerParam.middlewares, param.middlewares...)
		}
		if param.handler != nil {
			if node.handlerParam.handler != nil {
				panic("assign duplicated handler: " + subject)
			}
			node.handlerParam.handler = param.handler
		}
		return node
	}

	char := string(subject[idx])
	if node.child == nil {
		node.child = make(map[string]*trie[M])
	}

	next, exist := node.child[char]
	if !exist {
		next = &trie[M]{
			child:       nil,
			fullSubject: node.fullSubject + char,
		}
		node.child[char] = next
	}

	return next.insert(subject, idx+1, param)
}

func (node *trie[M]) handle(subject string, idx int, msg M) (err error) {
	if len(subject) == idx {
		return LinkMiddlewares(node.handler, node.middlewares...)(msg)
	}

	if node.transforms == nil {
		return node.handle(subject, idx+1, msg)
	}

	idx = 0
	for _, transform := range node.transforms {
		msg, err = transform(msg)
		if err != nil {
			return ErrorWrapWithMessage(err, "Failed to transform")
		}
	}

	subject, err = node.getSubject(msg)
	if err != nil {
		return ErrorWrapWithMessage(err, "Failed to parse subject from the message")
	}

	char := string(subject[idx])
	next, exist := node.child[char]
	if exist {
		return next.handle(subject, idx, msg)
	}

	if node.defaultHandler != nil {
		return LinkMiddlewares(node.defaultHandler, node.middlewares...)(msg)
	}

	if node.notFoundHandler != nil {
		return node.notFoundHandler(msg)
	}

	return ErrorWrapWithMessage(ErrNotFound, "mux subject")
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
