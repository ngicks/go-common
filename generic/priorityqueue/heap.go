package priorityqueue

import (
	"github.com/ngicks/go-common/generic/gcontainer/gheap"
)

// Heap[T] exposes all functions `container/heap` exposes as method on the type.
// Any structs which wish to expose heap functions as methods can embed *Heap[T, U] to them
// while still keeping all implementation details of T hidden.
type Heap[T any] struct {
	iface gheap.Interface[T]
}

func NewHeap[T any](iface gheap.Interface[T]) *Heap[T] {
	return &Heap[T]{
		iface: iface,
	}
}

func (hw *Heap[T]) Fix(i int) {
	gheap.Fix(hw.iface, i)
}
func (hw *Heap[T]) Init() {
	gheap.Init(hw.iface)
}
func (hw *Heap[T]) Pop() T {
	return gheap.Pop(hw.iface)
}
func (hw *Heap[T]) Push(x T) {
	gheap.Push(hw.iface, x)
}
func (hw *Heap[T]) Remove(i int) T {
	return gheap.Remove(hw.iface, i)
}

// SliceInterfaceMethods is replacement methods for SliceInterface[T].
// Non-nil fields will be used in heap functions of corresponding name.
type SliceInterfaceMethods[T any] struct {
	Swap func(s *[]T, i, j int)
	Push func(s *[]T, v T)
	Pop  func(s *[]T) T
}

var _ gheap.Interface[any] = (*SliceInterface[any])(nil)

// *SliceInterface[T] implements gheap.Interface[T].
// SliceInterface exposes its internal slice []T as Inner so that you can modify it as you wish.
type SliceInterface[T any] struct {
	Inner   []T
	less    func(i, j T) bool
	methods SliceInterfaceMethods[T]
}

// NewSliceInterface makes up a slice and less function into gheap.Interface[T].
// init is initial content of interface.
// If init is nil, new slice would be allocated and used.
// less must return true when i < j to make it a min heap. Or you can use more to be a a max heap.
// Non nil fields of methods will be used in corresponding method instead.
func NewSliceInterface[T any](
	init []T,
	less func(i, j T) bool,
	methods SliceInterfaceMethods[T],
) *SliceInterface[T] {
	if init == nil {
		init = make([]T, 0)
	}
	return &SliceInterface[T]{
		Inner:   init,
		less:    less,
		methods: methods,
	}
}

func (sl *SliceInterface[T]) Len() int {
	return len(sl.Inner)
}

func (sl *SliceInterface[T]) Less(i, j int) bool {
	return sl.less(sl.Inner[i], sl.Inner[j])
}

func (sl *SliceInterface[T]) Swap(i, j int) {
	if sl.methods.Swap != nil {
		sl.methods.Swap(&sl.Inner, i, j)
	} else {
		sl.Inner[i], sl.Inner[j] = sl.Inner[j], sl.Inner[i]
	}
}

func (sl *SliceInterface[T]) Push(x T) {
	if sl.methods.Push != nil {
		sl.methods.Push(&sl.Inner, x)
	} else {
		sl.Inner = append(sl.Inner, x)
	}
}

func (sl *SliceInterface[T]) Pop() T {
	if sl.methods.Pop != nil {
		return sl.methods.Pop(&sl.Inner)
	} else {
		if len(sl.Inner) == 0 {
			panic("github.com/ngicks/go-common/generic/priorityqueue: (*SliceInterface[T]).Pop() called on zero length slice")
		}
		v := sl.Inner[len(sl.Inner)-1]
		var zero T
		sl.Inner[len(sl.Inner)-1] = zero // zero out.
		sl.Inner = sl.Inner[:len(sl.Inner)-1]
		return v
	}
}
