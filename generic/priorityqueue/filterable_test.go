package heap

import (
	"slices"
	"testing"
)

func TestFilterable_with_additional_props(t *testing.T) {
	t.Run("Filter", func(t *testing.T) {
		h := NewFilterable(nil, func(i, j int) bool { return i < j }, SliceInterfaceMethods[int]{})

		h.Push(7)
		h.Push(4)
		h.Push(1)
		h.Push(6)
		h.Push(5)
		h.Push(3)
		h.Push(2)

		lenBefore := h.Len()
		h.Filter(func(s []int) []int { return slices.DeleteFunc(s, func(i int) bool { return i%2 == 0 }) })
		removed := lenBefore - h.Len()

		if removed != 3 {
			t.Fatalf("removed len must be %d, but is %d", 3, removed)
		}

		for i := 1; i <= 7; i = i + 2 {
			popped := h.Pop()
			if int(popped) != i {
				t.Errorf("pop = %v expected %v", popped, i)
			}
		}

		if h.Len() != 0 {
			t.Errorf("expect empty but size = %v", h.Len())
		}

		h.Push(7)
		h.Push(4)
		h.Push(1)
		h.Push(6)
		h.Push(5)
		h.Push(3)
		h.Push(2)

		lenBefore = h.Len()
		h.Filter(func(s []int) []int {
			part := slices.DeleteFunc(s[0:3], func(i int) bool { return i%2 == 0 })
			return slices.Delete(s, len(part), 3)
		})
		removed = lenBefore - h.Len()

		if removed != 1 {
			t.Fatalf("removed len must be %d, but is %d", 3, removed)
		}

		for h.Len() != 0 {
			h.Pop()
		}

		h.Push(7)
		h.Push(4)
		h.Push(1)
		h.Push(6)
		h.Push(5)
		h.Push(3)
		h.Push(2)

		lenBefore = h.Len()
		h.Filter(func(s []int) []int {
			part := slices.DeleteFunc(s[3:6], func(i int) bool { return i%2 == 0 })
			return slices.Delete(s, len(part)+3, 6)
		})
		removed = lenBefore - h.Len()
		if removed != 2 {
			t.Fatalf("removed len must be %d, but is %d", 3, removed)
		}
	})

	t.Run("Clone", func(t *testing.T) {
		h := NewFilterable(nil, func(i, j int) bool { return i < j }, SliceInterfaceMethods[int]{})

		h.Push(7)
		h.Push(4)
		h.Push(1)
		h.Push(6)
		h.Push(5)
		h.Push(3)
		h.Push(2)

		cloned := h.Clone()

		var out []int
		for h.Len() > 0 {
			out = append(out, h.Pop())
		}

		var outCloned []int
		for cloned.Len() > 0 {
			outCloned = append(outCloned, cloned.Pop())
		}

		for i := 0; i < len(out); i++ {
			if out[i] != outCloned[i] {
				t.Fatalf("not equal. expected = %d, actual = %d", out[i], outCloned[i])
			}
		}
	})
}

type item struct {
	value string
	index int
}

func TestFilterable_swap_is_used_if_implemented(t *testing.T) {
	h := NewFilterable[*item](
		nil,
		func(i, j *item) bool { return i.value < j.value },
		SliceInterfaceMethods[*item]{
			Swap: func(s *[]*item, i, j int) {
				(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
				(*s)[i].index = i
				(*s)[j].index = j
			},
			Push: func(s *[]*item, v *item) {
				v.index = len(*s)
				*s = append(*s, v)
			},
			Pop: func(s *[]*item) *item {
				v := (*s)[len(*s)-1]
				(*s)[len(*s)-1] = nil
				*s = (*s)[:len(*s)-1]
				v.index = -1
				return v
			},
		},
	)

	h.Push(&item{value: "foo"})
	h.Push(&item{value: "bar"})
	h.Push(&item{value: "baz"})
	h.Push(&item{value: "qux"})
	h.Push(&item{value: "quux"})

	slice := ([]*item)(h.iface.Inner)

	if ele := slice[0]; ele.index != 0 || ele.value != "bar" {
		t.Fatalf("incorrect: %+v", *ele)
	}
	if ele := slice[1]; ele.index != 1 || ele.value != "foo" {
		t.Fatalf("incorrect: %+v", *ele)
	}
	if ele := slice[2]; ele.index != 2 || ele.value != "baz" {
		t.Fatalf("incorrect: %+v", *ele)
	}
	if ele := slice[3]; ele.index != 3 || ele.value != "qux" {
		t.Fatalf("incorrect: %+v", *ele)
	}
	if ele := slice[4]; ele.index != 4 || ele.value != "quux" {
		t.Fatalf("incorrect: %+v", *ele)
	}

	for i := 0; i < 5; i++ {
		if item := h.Pop(); item.index != -1 {
			t.Fatalf("incorrect: %+v", *item)
		}
	}
}
