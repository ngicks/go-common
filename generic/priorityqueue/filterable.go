package priorityqueue

// Filterable[T] is a filterable priority queue made on top of []T.
type Filterable[T any] struct {
	*Heap[T]
	iface *SliceInterface[T]
}

// NewFilterable returns a new *Filterable[T].
// init is initial content for queue. If heap invariants are broken, call Init to re-establish them.
// less should return i < j for a min heap, otherwise return i > j for a max heap.
func NewFilterable[T any](init []T, less func(i, j T) bool, methods SliceInterfaceMethods[T]) *Filterable[T] {
	iface := NewSliceInterface(init, less, methods)
	h := NewHeap(iface)
	return &Filterable[T]{
		Heap:  h,
		iface: iface,
	}
}

// Peek returns most prioritized value in heap without removing it.
// If this heap contains 0 element, returned p is zero value for type T.
//
// The complexity is O(1).
func (h *Filterable[T]) Peek() (p T) {
	if len(h.iface.Inner) == 0 {
		return
	}
	return h.iface.Inner[0]
}

func (h *Filterable[T]) Len() int {
	return h.iface.Len()
}

// Clone clones internal slice and creates new FilterableHeap on it.
// This is done by simple slice copy, without succeeding Init call,
// which means it also clones broken invariants if any.
//
// If type T or one of its internal value is pointer type,
// mutation of T propagates cloned to original, and vice versa.
func (h *Filterable[T]) Clone() *Filterable[T] {
	cloned := make([]T, len(h.iface.Inner))
	copy(cloned, h.iface.Inner)

	n := NewFilterable[T](cloned, h.iface.less, h.iface.methods)
	return n
}

// Filter calls all filters and succeeding h.Init()
// to re-stablish heap state which should have been broken by filters.
//
// filters should not retain []T after they return.
//
// The complexity is at least O(n) where n is h.Len().
func (h *Filterable[T]) Filter(filters ...func(s []T) []T) {
	for _, fn := range filters {
		if fn != nil {
			h.iface.Inner = fn(h.iface.Inner)
		}
	}
	h.Init()
}
