package gsync

import (
	"sync"
)

type Pool[T any] struct {
	inner sync.Pool
}

func NewPool[T any](new func() T) *Pool[T] {
	p := &Pool[T]{}
	p.SetNew(new)
	return p
}

func (p *Pool[T]) SetNew(new func() T) {
	if new == nil {
		p.inner.New = nil
	} else {
		p.inner.New = func() any { return new() }
	}
}

func (p *Pool[T]) Get() T {
	got := p.inner.Get()
	if got == nil {
		var zero T
		return zero
	}
	return got.(T)
}

func (p *Pool[T]) Put(x T) {
	p.inner.Put(x)
}
