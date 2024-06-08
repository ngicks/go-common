package gsync

import (
	"sync"
)

type Map[K comparable, V any] struct {
	inner sync.Map
}

func (m *Map[K, V]) Delete(key K) {
	m.inner.Delete(key)
}

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	if v, ok := m.inner.Load(key); ok {
		return v.(V), true
	} else {
		var zero V
		return zero, false
	}
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.inner.LoadAndDelete(key)
	if v != nil {
		value = v.(V)
	}
	return value, loaded
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := m.inner.LoadOrStore(key, value)
	if v != nil {
		actual = v.(V)
	}
	return actual, loaded
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.inner.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (m *Map[K, V]) Store(key K, value V) {
	m.inner.Store(key, value)
}

func (m *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	if v, loaded := m.inner.Swap(key, value); loaded {
		return v.(V), true
	} else {
		var zero V
		return zero, false
	}
}

type ComparableMap[K, V comparable] struct {
	Map[K, V]
}

func (m *ComparableMap[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	return m.inner.CompareAndDelete(key, old)
}

func (m *ComparableMap[K, V]) CompareAndSwap(key K, old, new V) bool {
	return m.inner.CompareAndSwap(key, old, new)
}
