# generic

The package generic holds generic wrapper of Go's std library and some generic data structures.

Generic wrappers has `"g"` prefix in its package name.

## gcontainer

Generics wrappers of Go's standard `container` library

It only has a wrapper of `heap`, `gheap`.

Since `list` / `ring` make use of pointer addresses,
wrapping them effectively renders their interface unnatural and meaningless.

## gsync

Generic wrappers of Go's standard `sync` library.

## priorityqueue

The package priorityqueue implements some utility for the priority queue.

- The type `SliceInterface[T]` converts a less function and a slice `[]T` into `gheap.Interface[T]` and
- The type `Heap[T]` converts `gheap` functions into method sets
- And finally `Filterable[T]` implements a filterable priority queue based on slice `[]T`.
  - Its `Filter` method allows callers to manipulate queue content.
