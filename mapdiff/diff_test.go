package mapdiff

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name       string
		left       map[string]int
		right      map[string]int
		wantL      map[string]int
		wantCommon map[string]int
		wantDiff   map[string]Pair[int]
		wantR      map[string]int
	}{
		{
			name:       "identical maps",
			left:       map[string]int{"a": 1, "b": 2, "c": 3},
			right:      map[string]int{"a": 1, "b": 2, "c": 3},
			wantL:      map[string]int{},
			wantCommon: map[string]int{"a": 1, "b": 2, "c": 3},
			wantDiff:   map[string]Pair[int]{},
			wantR:      map[string]int{},
		},
		{
			name:       "completely different maps",
			left:       map[string]int{"a": 1, "b": 2},
			right:      map[string]int{"c": 3, "d": 4},
			wantL:      map[string]int{"a": 1, "b": 2},
			wantCommon: map[string]int{},
			wantDiff:   map[string]Pair[int]{},
			wantR:      map[string]int{"c": 3, "d": 4},
		},
		{
			name:       "overlapping with different values",
			left:       map[string]int{"a": 1, "b": 2, "c": 3},
			right:      map[string]int{"b": 5, "c": 3, "d": 4},
			wantL:      map[string]int{"a": 1},
			wantCommon: map[string]int{"c": 3},
			wantDiff:   map[string]Pair[int]{"b": {L: 2, R: 5}},
			wantR:      map[string]int{"d": 4},
		},
		{
			name:       "empty left map",
			left:       map[string]int{},
			right:      map[string]int{"a": 1, "b": 2},
			wantL:      map[string]int{},
			wantCommon: map[string]int{},
			wantDiff:   map[string]Pair[int]{},
			wantR:      map[string]int{"a": 1, "b": 2},
		},
		{
			name:       "empty right map",
			left:       map[string]int{"a": 1, "b": 2},
			right:      map[string]int{},
			wantL:      map[string]int{"a": 1, "b": 2},
			wantCommon: map[string]int{},
			wantDiff:   map[string]Pair[int]{},
			wantR:      map[string]int{},
		},
		{
			name:       "both empty maps",
			left:       map[string]int{},
			right:      map[string]int{},
			wantL:      map[string]int{},
			wantCommon: map[string]int{},
			wantDiff:   map[string]Pair[int]{},
			wantR:      map[string]int{},
		},
		{
			name:       "mixed overlap scenario",
			left:       map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5},
			right:      map[string]int{"b": 2, "c": 6, "d": 4, "f": 7, "g": 8},
			wantL:      map[string]int{"a": 1, "e": 5},
			wantCommon: map[string]int{"b": 2, "d": 4},
			wantDiff:   map[string]Pair[int]{"c": {L: 3, R: 6}},
			wantR:      map[string]int{"f": 7, "g": 8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotL, gotCommon, gotDiff, gotR := Diff(tt.left, tt.right)

			if !reflect.DeepEqual(gotL, tt.wantL) {
				t.Errorf("Diff() gotL = %v, want %v", gotL, tt.wantL)
			}
			if !reflect.DeepEqual(gotCommon, tt.wantCommon) {
				t.Errorf("Diff() gotCommon = %v, want %v", gotCommon, tt.wantCommon)
			}
			if !reflect.DeepEqual(gotDiff, tt.wantDiff) {
				t.Errorf("Diff() gotDiff = %v, want %v", gotDiff, tt.wantDiff)
			}
			if !reflect.DeepEqual(gotR, tt.wantR) {
				t.Errorf("Diff() gotR = %v, want %v", gotR, tt.wantR)
			}
		})
	}
}

func TestDiffFunc(t *testing.T) {
	// Test with case-insensitive string comparison
	caseInsensitiveCmp := func(s1, s2 string) bool {
		return strings.EqualFold(s1, s2)
	}

	tests := []struct {
		name       string
		left       map[string]string
		right      map[string]string
		cmp        func(string, string) bool
		wantL      map[string]string
		wantCommon map[string]string
		wantDiff   map[string]Pair[string]
		wantR      map[string]string
	}{
		{
			name:       "case insensitive comparison - all common",
			left:       map[string]string{"a": "Hello", "b": "World"},
			right:      map[string]string{"a": "hello", "b": "WORLD"},
			cmp:        caseInsensitiveCmp,
			wantL:      map[string]string{},
			wantCommon: map[string]string{"a": "Hello", "b": "World"},
			wantDiff:   map[string]Pair[string]{},
			wantR:      map[string]string{},
		},
		{
			name:       "case insensitive comparison - some different",
			left:       map[string]string{"a": "Hello", "b": "World", "c": "Test"},
			right:      map[string]string{"a": "hello", "b": "Earth", "d": "Example"},
			cmp:        caseInsensitiveCmp,
			wantL:      map[string]string{"c": "Test"},
			wantCommon: map[string]string{"a": "Hello"},
			wantDiff:   map[string]Pair[string]{"b": {L: "World", R: "Earth"}},
			wantR:      map[string]string{"d": "Example"},
		},
		{
			name:  "length comparison",
			left:  map[string]string{"a": "short", "b": "medium", "c": "longtext", "e": "test"},
			right: map[string]string{"a": "small", "b": "middle", "c": "huge", "d": "x"},
			cmp: func(s1, s2 string) bool {
				return len(s1) == len(s2)
			},
			wantL:      map[string]string{"e": "test"},
			wantCommon: map[string]string{"a": "short", "b": "medium"},
			wantDiff:   map[string]Pair[string]{"c": {L: "longtext", R: "huge"}},
			wantR:      map[string]string{"d": "x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotL, gotCommon, gotDiff, gotR := DiffFunc(tt.left, tt.right, tt.cmp)

			if !reflect.DeepEqual(gotL, tt.wantL) {
				t.Errorf("DiffFunc() gotL = %v, want %v", gotL, tt.wantL)
			}
			if !reflect.DeepEqual(gotCommon, tt.wantCommon) {
				t.Errorf("DiffFunc() gotCommon = %v, want %v", gotCommon, tt.wantCommon)
			}
			if !reflect.DeepEqual(gotDiff, tt.wantDiff) {
				t.Errorf("DiffFunc() gotDiff = %v, want %v", gotDiff, tt.wantDiff)
			}
			if !reflect.DeepEqual(gotR, tt.wantR) {
				t.Errorf("DiffFunc() gotR = %v, want %v", gotR, tt.wantR)
			}
		})
	}
}

// Test with custom types
type CustomStruct struct {
	ID    int
	Value string
}

func TestDiffWithCustomTypes(t *testing.T) {
	left := map[string]CustomStruct{
		"key1": {ID: 1, Value: "one"},
		"key2": {ID: 2, Value: "two"},
		"key3": {ID: 3, Value: "three"},
	}

	right := map[string]CustomStruct{
		"key2": {ID: 2, Value: "two"},
		"key3": {ID: 3, Value: "modified"},
		"key4": {ID: 4, Value: "four"},
	}

	gotL, gotCommon, gotDiff, gotR := Diff(left, right)

	expectedL := map[string]CustomStruct{
		"key1": {ID: 1, Value: "one"},
	}
	expectedCommon := map[string]CustomStruct{
		"key2": {ID: 2, Value: "two"},
	}
	expectedDiff := map[string]Pair[CustomStruct]{
		"key3": {
			L: CustomStruct{ID: 3, Value: "three"},
			R: CustomStruct{ID: 3, Value: "modified"},
		},
	}
	expectedR := map[string]CustomStruct{
		"key4": {ID: 4, Value: "four"},
	}

	if !reflect.DeepEqual(gotL, expectedL) {
		t.Errorf("Diff() with custom types gotL = %v, want %v", gotL, expectedL)
	}
	if !reflect.DeepEqual(gotCommon, expectedCommon) {
		t.Errorf("Diff() with custom types gotCommon = %v, want %v", gotCommon, expectedCommon)
	}
	if !reflect.DeepEqual(gotDiff, expectedDiff) {
		t.Errorf("Diff() with custom types gotDiff = %v, want %v", gotDiff, expectedDiff)
	}
	if !reflect.DeepEqual(gotR, expectedR) {
		t.Errorf("Diff() with custom types gotR = %v, want %v", gotR, expectedR)
	}
}

// Benchmark tests
func BenchmarkDiff(b *testing.B) {
	left := make(map[string]int)
	right := make(map[string]int)

	// Create test data
	for i := range 1000 {
		if i%2 == 0 {
			left[string(rune('a'+i%26))+string(rune(i))] = i
		}
		if i%3 == 0 {
			right[string(rune('a'+i%26))+string(rune(i))] = i * 2
		}
	}

	for b.Loop() {
		Diff(left, right)
	}
}

func BenchmarkDiffFunc(b *testing.B) {
	left := make(map[string]string)
	right := make(map[string]string)

	// Create test data
	for i := range 1000 {
		if i%2 == 0 {
			left[string(rune('a'+i%26))+string(rune(i))] = "value" + string(rune(i))
		}
		if i%3 == 0 {
			right[string(rune('a'+i%26))+string(rune(i))] = "VALUE" + string(rune(i))
		}
	}

	cmp := func(s1, s2 string) bool {
		return strings.EqualFold(s1, s2)
	}

	for b.Loop() {
		DiffFunc(left, right, cmp)
	}
}

