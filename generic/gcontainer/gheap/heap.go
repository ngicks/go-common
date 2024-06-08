package gheap

import (
	"container/heap"
	"sort"
)

// Interface is same as `container/heap`.Interface but uses type-param
type Interface[T any] interface {
	sort.Interface
	Push(x T)
	Pop() T
}

// interfaceReverter is reverter that widens T to any.
type interfaceReverter[T any] struct {
	Interface[T]
}

func (ir interfaceReverter[T]) Push(x any) {
	ir.Interface.Push(x.(T))
}

func (ir interfaceReverter[T]) Pop() any {
	return ir.Interface.Pop()
}

// Init wraps h to make it compatible with `container/heap` then call `container/heap`.Init with it.
func Init[T any](h Interface[T]) {
	heap.Init(interfaceReverter[T]{h})
}

// Push wraps h to make it compatible with `container/heap` then call `container/heap`.Push with it.
func Push[T any](h Interface[T], x T) {
	heap.Push(interfaceReverter[T]{h}, x)
}

// Pop wraps h to make it compatible with `container/heap` then call `container/heap`.Pop with it.
// If internal Pop returns nil value, this Pop returns zero-value for T.
func Pop[T any](h Interface[T]) (v T) {
	if popped := heap.Pop(interfaceReverter[T]{h}); popped != nil {
		// Pop'ping empty Interface may or may not panic.
		return popped.(T)
	} else {
		return
	}
}

// Remove wraps h to make it compatible with `container/heap` then call `container/heap`.Remove with it.
// If internal Remove returns nil value, this Remove returns zero-value for T.
func Remove[T any](h Interface[T], i int) (v T) {
	if removed := heap.Remove(interfaceReverter[T]{h}, i); removed != nil {
		// Remove'ing empty Interface may or may not panic.
		return removed.(T)
	} else {
		return
	}
}

// Fix wraps h to make it compatible with `container/heap` then call `container/heap`.Fix with it.
func Fix[T any](h Interface[T], i int) {
	heap.Fix(interfaceReverter[T]{h}, i)
}
