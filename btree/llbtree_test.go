package btree

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/ngicks/go-common/btree/internal/hiter"
)

type keyValue[K, V any] struct {
	K K
	V V
}

var logCollect2 = false

func collect2[K, V any, K2 comparable](t *testing.T, uniq func(K) K2, seq iter.Seq2[K, V]) []keyValue[K, V] {
	var seen map[K2]bool
	if t != nil {
		seen = make(map[K2]bool)
		t.Helper()
	}

	var kv []keyValue[K, V]
	for k, v := range seq {
		if t != nil {
			if seen[uniq(k)] {
				t.Errorf("loop detected!: %v", k)
				return kv
			} else {
				seen[uniq(k)] = true
				if logCollect2 {
					t.Logf("k = %v, v = %v", k, v)
				}
			}
		}
		kv = append(kv, keyValue[K, V]{k, v})
	}
	return kv
}

func values2[K, V any](kv []keyValue[K, V]) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, kv := range kv {
			if !yield(kv.K, kv.V) {
				return
			}
		}
	}
}

var llbTreeExpected = []keyValue[int, string]{
	{0, "foo"},
	{2, "bar"},
	{4, "baz"},
	{5, "qux"},
	{7, "quux"},
}

func omitL[K, V any](seq iter.Seq2[K, V]) iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range seq {
			if !yield(k) {
				return
			}
		}
	}
}

func runInAllPattern[V any](t *testing.T, patterns []V, fn func(t *testing.T, pattern []V)) {
	t.Helper()

	for pat := range hiter.Permutations(slices.Clone(patterns)) {
		var keys strings.Builder
		for i, v := range pat {
			if i > 0 {
				keys.WriteByte(',')
			}
			fmt.Fprintf(&keys, "%v", v)
		}

		t.Run(
			keys.String(),
			func(t *testing.T) {
				t.Helper()
				fn(t, pat)
			},
		)
	}
}

func runInAllPattern2[K, V any](t *testing.T, patterns []keyValue[K, V], fn func(t *testing.T, pattern []keyValue[K, V])) {
	t.Helper()

	for pat := range hiter.Permutations(slices.Clone(patterns)) {
		var keys strings.Builder
		for i, kv := range pat {
			if i > 0 {
				keys.WriteByte(',')
			}
			fmt.Fprintf(&keys, "%v", kv.K)
		}

		t.Run(
			keys.String(),
			func(t *testing.T) {
				t.Helper()
				fn(t, pat)
			},
		)
	}
}

func TestLLBTree_All(t *testing.T) {
	runInAllPattern2(
		t,
		llbTreeExpected,
		func(t *testing.T, pattern []keyValue[int, string]) {
			tree := NewLLBTreeOrdered[int, string]()
			tree.InsertSeq(values2(pattern))

			collected := collect2(t, func(i int) int { return i }, tree.All())

			if !slices.Equal(llbTreeExpected, collected) {
				t.Fatalf("not as expected = %#v", collected)
			}
		},
	)
}

func TestLLBTree_Backward(t *testing.T) {
	runInAllPattern2(
		t,
		llbTreeExpected,
		func(t *testing.T, pattern []keyValue[int, string]) {
			tree := NewLLBTreeOrdered[int, string]()
			tree.InsertSeq(values2(pattern))

			collected := collect2(t, func(i int) int { return i }, tree.Backward())

			expected := slices.Clone(llbTreeExpected)
			slices.Reverse(expected)
			if !slices.Equal(expected, collected) {
				t.Fatalf("not as expected = %#v", collected)
			}
		},
	)
}

func TestLLBTree_Min(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		tree := NewLLBTreeOrdered[int, string]()

		if _, _, ok := tree.Min(); ok {
			t.Fatalf("wrong min")
		}

		expected := slices.Clone(llbTreeExpected)
		slices.Reverse(expected)
		for k, v := range values2(expected) {
			tree.Insert(k, v)
			if k2, _, ok := tree.Min(); !ok || k2 != k {
				t.Fatalf("wrong min: %d, %d", k2, k)
			}
		}
	})
	t.Run("no update", func(t *testing.T) {
		runInAllPattern2(
			t,
			llbTreeExpected,
			func(t *testing.T, pattern []keyValue[int, string]) {
				tree := NewLLBTreeOrdered[int, string]()

				if _, _, ok := tree.Min(); ok {
					t.Fatalf("wrong min")
				}

				tree.Insert(0, "000")

				if k, _, ok := tree.Min(); !ok || k != 0 {
					t.Fatalf("wrong min")
				}

				for k, v := range values2(pattern) {
					tree.Insert(k, v)
					if k, _, ok := tree.Min(); !ok || k != 0 {
						t.Fatalf("wrong min")
					}
				}
			},
		)
	})
}

func TestLLBTree_Max(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		tree := NewLLBTreeOrdered[int, string]()

		if _, _, ok := tree.Max(); ok {
			t.Fatalf("wrong max")
		}

		for k, v := range values2(llbTreeExpected) {
			tree.Insert(k, v)
			if k2, _, ok := tree.Max(); !ok || k2 != k {
				t.Fatalf("wrong max: %d, %d", k2, k)
			}
		}
	})
	t.Run("no update", func(t *testing.T) {
		runInAllPattern2(
			t,
			llbTreeExpected,
			func(t *testing.T, pattern []keyValue[int, string]) {
				tree := NewLLBTreeOrdered[int, string]()

				if _, _, ok := tree.Max(); ok {
					t.Fatalf("wrong max")
				}

				tree.Insert(7, "000")

				if k, _, ok := tree.Max(); !ok || k != 7 {
					t.Fatalf("wrong max")
				}

				for k, v := range values2(pattern) {
					tree.Insert(k, v)
					if k, _, ok := tree.Max(); !ok || k != 7 {
						t.Fatalf("wrong max")
					}
				}
			},
		)
	})
}

func TestLLBTree_Scan(t *testing.T) {
	runInAllPattern2(
		t,
		llbTreeExpected,
		func(t *testing.T, pattern []keyValue[int, string]) {
			t.Helper()

			tree := NewLLBTreeOrdered[int, string]()
			tree.InsertSeq(values2(pattern))

			assertEqual := func(t *testing.T, expected, actual []keyValue[int, string]) {
				t.Helper()
				if !slices.Equal(expected, actual) {
					t.Errorf(
						"not equal:\nexpected: %#v\nactual  : %#v",
						expected, actual,
					)
				}
			}

			assertEqual(
				t,
				[]keyValue[int, string]{{4, "baz"}},
				collect2(t, func(i int) int { return i }, tree.Scan(3, 4)),
			)
			assertEqual(
				t,
				[]keyValue[int, string]{{2, "bar"}},
				collect2(t, func(i int) int { return i }, tree.Scan(2, 3)),
			)
			assertEqual(
				t,
				[]keyValue[int, string]{},
				collect2(t, func(i int) int { return i }, tree.Scan(3, 3)),
			)

			assertEqual(
				t,
				[]keyValue[int, string]{
					{2, "bar"},
					{4, "baz"},
					{5, "qux"},
				},
				collect2(t, func(i int) int { return i }, tree.Scan(2, 5)),
			)

			assertEqual(
				t,
				[]keyValue[int, string]{
					{5, "qux"},
					{4, "baz"},
					{2, "bar"},
				},
				collect2(t, func(i int) int { return i }, tree.Scan(5, 1)),
			)
		},
	)
}

func TestLLBTree_remove_single_element(t *testing.T) {
	runInAllPattern2(
		t,
		llbTreeExpected,
		func(t *testing.T, pattern []keyValue[int, string]) {
			for i := range values2(llbTreeExpected) {
				t.Run(fmt.Sprintf("remove %dth", i), func(t *testing.T) {
					tree := NewLLBTreeOrdered[int, string]()
					tree.InsertSeq(values2(pattern))

					removed := tree.Remove(i)
					t.Logf("removed %d", i)
					if !removed {
						t.Fatalf("tried to Remove %d, but Remove returned false", i)
					}

					expected2 := slices.Clone(llbTreeExpected)
					expected2 = slices.DeleteFunc(
						expected2,
						func(kv keyValue[int, string]) bool {
							return kv.K == i
						},
					)

					collected := collect2(t, func(i int) int { return i }, tree.All())
					if !slices.Equal(expected2, collected) {
						t.Fatalf("not as expected = %#v", collected)
					}
				})
			}
		},
	)
}

func TestLLBTree_remove_all_elements(t *testing.T) {
	runInAllPattern2(
		t,
		llbTreeExpected,
		func(t *testing.T, pattern []keyValue[int, string]) {
			runInAllPattern(
				t,
				slices.Collect(omitL(values2(pattern))),
				func(t *testing.T, indices []int) {
					tree := NewLLBTreeOrdered[int, string]()
					tree.InsertSeq(values2(pattern))

					for _, i := range indices {
						removed := tree.Remove(i)
						t.Logf("removed %d", i)
						if !removed {
							t.Fatalf("tried to Remove %d, but Remove returned false", i)
						}
					}

					collected := collect2(t, func(i int) int { return i }, tree.All())

					if len(collected) != 0 {
						t.Fatalf("not as expected = %#v", collected)
					}
				},
			)
		},
	)
}

func TestLLBTree_remove_while_iteration(t *testing.T) {
	runInAllPattern2(
		t,
		llbTreeExpected,
		func(t *testing.T, pattern []keyValue[int, string]) {
			for i := range values2(llbTreeExpected) {
				t.Run(fmt.Sprintf("remove %dth", i), func(t *testing.T) {
					tree := NewLLBTreeOrdered[int, string]()
					tree.InsertSeq(values2(pattern))

					var collected []keyValue[int, string]
					for k, v := range tree.All() {
						if k == i {
							removed := tree.Remove(i)
							t.Logf("removed %d", i)
							if !removed {
								t.Fatalf("tried to Remove %d, but Remove returned false", i)
							}
						}
						collected = append(collected, keyValue[int, string]{k, v})
					}

					if !slices.Equal(llbTreeExpected, collected) {
						t.Fatalf("not as expected = %#v", collected)
					}
				})
			}
		},
	)
}
