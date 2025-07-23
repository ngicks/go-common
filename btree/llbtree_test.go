package btree

import (
	"iter"
	"math/rand/v2"
	"slices"
	"testing"
)

type keyValue[K, V any] struct {
	K K
	V V
}

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
				t.Logf("k = %v, v = %v", k, v)
			}
		}
		kv = append(kv, keyValue[K, V]{k, v})
	}
	return kv
}

func TestLLBTree(t *testing.T) {
	expected := []keyValue[int, string]{
		{0, "foo"},
		{1, "bar"},
		{2, "baz"},
		{3, "qux"},
		{4, "quux"},
	}

	cloneShuffle := func() []keyValue[int, string] {
		in := slices.Clone(expected)
		rand.Shuffle(len(in), func(i, j int) { in[i], in[j] = in[j], in[i] })
		return in
	}

	t.Run("new", func(t *testing.T) {
		// t.Skip()

		tree := NewLLBTreeOrdered[int, string]()

		for _, kv := range cloneShuffle() {
			t.Logf("inserting %d:%s", kv.K, kv.V)
			tree.insert(kv.K, kv.V)
		}

		collected := collect2(t, func(i int) int { return i }, tree.all())
		if !slices.Equal(expected, collected) {
			t.Fatalf("not as expected = %#v", collected)
		}
	})

	t.Run("delete single ele", func(t *testing.T) {
		// t.Skip()

		tree := NewLLBTreeOrdered[int, string]()

		for _, kv := range cloneShuffle() {
			t.Logf("inserting %d:%s", kv.K, kv.V)
			tree.insert(kv.K, kv.V)
		}

		i := rand.N(len(expected))

		deleted := tree.delete(i)
		t.Logf("deleted %d", i)
		if !deleted {
			t.Fatalf("tried to delete %d, but delete returned false", i)
		}

		expected2 := slices.Clone(expected)
		expected2 = slices.Delete(expected2, i, i+1)

		collected := collect2(t, func(i int) int { return i }, tree.all())
		if !slices.Equal(expected2, collected) {
			t.Fatalf("not as expected = %#v", collected)
		}
	})

	t.Run("delete all ele", func(t *testing.T) {
		// t.Skip()
		tree := NewLLBTreeOrdered[int, string]()
		for _, kv := range cloneShuffle() {
			tree.insert(kv.K, kv.V)
		}

		indices := []int{0, 1, 2, 3, 4}
		rand.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })

		for _, i := range indices {
			deleted := tree.delete(i)
			t.Logf("deleted %d", i)
			if !deleted {
				t.Fatalf("tried to delete %d, but delete returned false", i)
			}
		}

		collected := collect2(t, func(i int) int { return i }, tree.all())
		if len(collected) != 0 {
			t.Fatalf("not as expected = %#v", collected)
		}
	})

	t.Run("delete while iterate", func(t *testing.T) {
		// t.Skip()

		tree := NewLLBTreeOrdered[int, string]()

		for _, kv := range cloneShuffle() {
			t.Logf("inserting %d:%s", kv.K, kv.V)
			tree.insert(kv.K, kv.V)
		}

		i := rand.N(len(expected))

		var collected []keyValue[int, string]
		for k, v := range tree.all() {
			if k == i {
				deleted := tree.delete(i)
				t.Logf("deleted %d", i)
				if !deleted {
					t.Fatalf("tried to delete %d, but delete returned false", i)
				}
			}
			collected = append(collected, keyValue[int, string]{k, v})
		}
		if !slices.Equal(expected, collected) {
			t.Fatalf("not as expected = %#v", collected)
		}
	})
}
