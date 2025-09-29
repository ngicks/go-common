// Package mapdiff has some functions to diff 2 map[K]V.
package mapdiff

import "maps"

func Diff[M ~map[K]V, K, V comparable](l1, r1 M) (l M, common M, diff map[K]Pair[V], r M) {
	return DiffFunc(l1, r1, func(v1, v2 V) bool { return v1 == v2 })
}

type Pair[V any] struct {
	L, R V
}

func DiffFunc[M ~map[K]V, K comparable, V any](l1, r1 M, cmp func(v1, v2 V) bool) (l M, common M, diff map[K]Pair[V], r M) {
	l = make(M)
	common = make(M)
	diff = make(map[K]Pair[V])
	r = maps.Clone(r1)
	for k := range l1 {
		lv := l1[k]
		rv, ok := r1[k]
		if !ok {
			l[k] = lv
			continue
		}
		delete(r, k)
		if cmp(lv, rv) {
			common[k] = lv
		} else {
			diff[k] = Pair[V]{lv, rv}
		}
	}
	return
}
