package contextkey

import (
	"log/slog"
	"slices"
	"sync"
)

// equalSyncMap is a helper function for generated test.
func equalSyncMap(v1, v2 any) bool {
	m1, ok := v1.(*sync.Map)
	if !ok {
		return false
	}
	m2, ok := v2.(*sync.Map)
	if !ok {
		return false
	}
	if m1 == nil || m2 == nil {
		return m1 == m2
	}
	eq := true
	var keys []any
	m1.Range(func(key, v1 any) bool {
		keys = append(keys, key)
		v2, ok := m2.Load(key)
		if !ok || v2 != v1 {
			eq = false
			return false
		}
		return true
	})
	if !eq {
		return false
	}
	m2.Range(func(key, value any) bool {
		if !slices.Contains(keys, key) {
			eq = false
			return false
		}
		return true
	})
	return eq
}

func equalSlogAttrs(v1, v2 any) bool {
	attrs1, ok := v1.([]slog.Attr)
	if !ok {
		return false
	}
	attrs2, ok := v2.([]slog.Attr)
	if !ok {
		return false
	}

	return slices.EqualFunc(attrs1, attrs2, func(l, r slog.Attr) bool {
		return l.Equal(r)
	})
}
