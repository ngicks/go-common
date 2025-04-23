package exver

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
	"strconv"
	"testing"
)

func TestNewCore(t *testing.T) {
	bad := [][]uint16{
		{},
		{1, 2, 3, 4, 5},
		{10000},
	}
	for _, tc := range bad {
		_, err := NewCore(tc)
		if err == nil {
			t.Errorf("%q: must return non-nil error", tc)
		}
	}

	good := []struct {
		in  []uint16
		out Core
	}{
		{[]uint16{1}, Core{component: [4]uint16{1}, length: 1}},
		{[]uint16{1, 2}, Core{component: [4]uint16{1, 2}, length: 2}},
		{[]uint16{1, 2, 3}, Core{component: [4]uint16{1, 2, 3}, length: 3}},
		{[]uint16{1, 2, 3, 4}, Core{component: [4]uint16{1, 2, 3, 4}, length: 4}},
		{[]uint16{99, 888, 777, 6666}, Core{component: [4]uint16{99, 888, 777, 6666}, length: 4}},
	}

	for _, tc := range good {
		c, err := NewCore(tc.in)
		if err != nil {
			t.Errorf("%q: must not return non-nil error(%q)", tc.in, err)
		}
		if c != tc.out {
			t.Errorf("%q: not equal\nexpected = %#v\nactual = %#v", tc.in, tc.out, c)
		}
	}
}

func TestParseCore(t *testing.T) {
	bad := []string{
		"bad",
		"1.2.3.4.5",
		"1.2-foo",
		"1.2.3-foo",
		"1.2.3foo",
		"01.02.03",
		"aa.bb.cc",
		"10000.10000",
	}
	for _, tc := range bad {
		_, err := ParseCore(tc)
		if err == nil {
			t.Errorf("%q: ParseCore must return error", tc)
		}
	}

	good := []struct {
		in  string
		out Core
	}{
		{"1", Core{component: [4]uint16{1}, length: 1}},
		{"1.2", Core{component: [4]uint16{1, 2}, length: 2}},
		{"1.2.3", Core{component: [4]uint16{1, 2, 3}, length: 3}},
		{"1.2.3.4", Core{component: [4]uint16{1, 2, 3, 4}, length: 4}},
		{"99.888.777.6666", Core{component: [4]uint16{99, 888, 777, 6666}, length: 4}},
	}

	for _, tc := range good {
		c, err := ParseCore(tc.in)
		if err != nil {
			t.Errorf("%q: must not return non-nil error(%q)", tc.in, err)
		}
		if c != tc.out {
			t.Errorf("%q: not equal\nexpected = %#v\nactual = %#v", tc.in, tc.out, c)
		}
	}
}

func TestCore_Component(t *testing.T) {
	var c Core
	c, _ = NewCore([]uint16{1, 2, 3})
	if c.Component() != [4]uint16{1, 2, 3, 0} {
		t.Errorf("not equal:\nexpected = %#v\nactual = %#v", [4]uint16{1, 2, 3, 0}, c.Component())
	}
}

func TestCore_Major_Minor_Patch_Extra(t *testing.T) {
	var (
		c   Core
		set [4]uint16
	)
	c, _ = NewCore([]uint16{1})
	set = [4]uint16{c.Major(), c.Minor(), c.Patch(), c.Extra()}
	if set != [4]uint16{1, 0, 0, 0} {
		t.Errorf("not equal:\nexpected = %#v\nactual = %#v", [4]uint16{1, 0, 0, 0}, set)
	}
	c, _ = NewCore([]uint16{4, 3, 2, 1})
	set = [4]uint16{c.Major(), c.Minor(), c.Patch(), c.Extra()}
	if set != [4]uint16{4, 3, 2, 1} {
		t.Errorf("not equal:\nexpected = %#v\nactual = %#v", [4]uint16{4, 3, 2, 1}, set)
	}
}

func TestCore_Normalize_NormalizeExtended(t *testing.T) {
	for _, tc := range []struct {
		in                         Core
		expected, expectedExtended string
	}{
		{
			MustNewCore([]uint16{1}),
			"1.0.0",
			"1.0.0.0",
		},
		{
			MustNewCore([]uint16{4, 3, 2, 1}),
			"4.3.2",
			"4.3.2.1",
		},
	} {
		if tc.in.Normalize().String() != tc.expected {
			t.Errorf("not equal:\nexpected = %s\nactual = %s", tc.expected, tc.in.Normalize().String())
		}
		if tc.in.NormalizeExtended().String() != tc.expectedExtended {
			t.Errorf("not equal:\nexpected = %s\nactual = %s", tc.expectedExtended, tc.in.NormalizeExtended().String())
		}
	}
}

func TestCore_Nums(t *testing.T) {
	for _, tc := range []struct {
		in       []uint16
		expected []uint
	}{
		{[]uint16{1}, []uint{1}},
		{[]uint16{1, 2}, []uint{1, 2}},
		{[]uint16{1, 2, 3, 4}, []uint{1, 2, 3, 4}},
	} {
		c, _ := NewCore(tc.in)
		if !slices.Equal(c.Nums(), tc.expected) {
			t.Errorf("not equal:\nexpected = %#v\nactual = %#v", tc.expected, c.Nums())
		}
	}
}

func TestCore_Int64(t *testing.T) {
	for _, tc := range []struct {
		in       []uint16
		expected int64
	}{
		{[]uint16{1}, 1_0000_0000_0000},
		{[]uint16{1, 2}, 1_0002_0000_0000},
		{[]uint16{1, 2, 3, 4}, 1_0002_0003_0004},
		{[]uint16{9999, 9999, 9999, 9999}, 9999_9999_9999_9999},
	} {
		c, _ := NewCore(tc.in)
		if c.Int64() != tc.expected {
			t.Errorf("not equal:\nexpected = %d\nactual = %d", tc.expected, c.Int64())
		}
	}
}

func TestCore_String_UnmarshalText_MarshalText(t *testing.T) {
	for _, tc := range []struct {
		in       []uint16
		expected string
	}{
		{[]uint16{}, "0.0.0"},
		{[]uint16{1}, "1"},
		{[]uint16{1, 2}, "1.2"},
		{[]uint16{1, 2, 3, 4}, "1.2.3.4"},
		{[]uint16{9999, 9999, 9999, 9999}, "9999.9999.9999.9999"},
	} {
		c, _ := NewCore(tc.in)
		if c.String() != tc.expected {
			t.Errorf("not equal:\nexpected = %s\nactual = %s", tc.expected, c.String())
		}
		if bin, _ := c.MarshalText(); string(bin) != tc.expected {
			t.Errorf("not equal:\nexpected = %s\nactual = %s", tc.expected, bin)
		}
		var c2 Core
		err := c2.UnmarshalText([]byte(tc.expected))
		if err != nil {
			t.Errorf("must be nil: %v", err)
		}
		if c.String() != c2.String() {
			t.Errorf("not equal:\nexpected = %s\nactual = %s", c.String(), c2.String())
		}
	}
}

func TestCore_UnmarshalJSON_MarshalJSON(t *testing.T) {
	bad := []string{
		"",
		"\"\"",
		"foo.bar",
	}

	for _, tc := range bad {
		var c Core
		err := c.UnmarshalJSON([]byte(tc))
		if err == nil {
			t.Errorf("%q: must not be nil", tc)
		}
	}

	good := []struct {
		in  string
		out Core
	}{
		{"\"1.0.0\"", Core{component: [4]uint16{1}, length: 3}},
		{"\"0.0.0\"", Core{component: [4]uint16{}, length: 3}},
		{"\"1.2.3.4\"", Core{component: [4]uint16{1, 2, 3, 4}, length: 4}},
	}
	for _, tc := range good {
		var c Core
		err := c.UnmarshalJSON([]byte(tc.in))
		if err != nil {
			t.Errorf("must be nil: %v", err)
		}
		if tc.out != c {
			t.Errorf("not equal:\nexpected = %#v\nactual = %#v", tc.out, c)
		}
		if marshaled, _ := c.MarshalJSON(); string(marshaled) != tc.in {
			t.Errorf("not equal:\nexpected = %s\nactual = %s", tc.in, string(marshaled))
		}
	}
}

type parseTestCase struct {
	in  string
	out *Version
}

var tests = []parseTestCase{
	{"bad", nil},
	{"v1-alpha.beta.gamma", nil},
	{"v1-pre", nil},
	{"v1+meta", nil},
	{"v1-pre+meta", nil},
	{"v1.2-pre", nil},
	{"v1.2+meta", nil},
	{"v1.2-pre+meta", nil},
	{"v1.0.0-alpha", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "alpha", build: ""}},
	{"v1.0.0-alpha.1", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "alpha.1", build: ""}},
	{"v1.0.0-alpha.beta", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "alpha.beta", build: ""}},
	{"v1.0.0-beta", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "beta", build: ""}},
	{"v1.0.0-beta.2", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "beta.2", build: ""}},
	{"v1.0.0-beta.11", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "beta.11", build: ""}},
	{"v1.0.0-rc.1", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "rc.1", build: ""}},
	{"v1", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 1}, prerelease: "", build: ""}},
	{"v1.0", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 2}, prerelease: "", build: ""}},
	{"v1.0.0", &Version{vPrefix: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "", build: ""}},
	{"v1.2", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2}, length: 2}, prerelease: "", build: ""}},
	{"v1.2.0", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2}, length: 3}, prerelease: "", build: ""}},
	{"v1.2.3-456", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456", build: ""}},
	{"v1.2.3-456.789", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456.789", build: ""}},
	{"v1.2.3-456-789", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456-789", build: ""}},
	{"v1.2.3-456a", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456a", build: ""}},
	{"v1.2.3-pre", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "pre", build: ""}},
	{"v1.2.3-pre+meta", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "pre", build: "meta"}},
	{"v1.2.3-pre.1", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "pre.1", build: ""}},
	{"v1.2.3-zzz", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "zzz", build: ""}},
	{"v1.2.3", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: ""}},
	{"v1.2.3+meta", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: "meta"}},
	{"v1.2.3+meta-pre", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: "meta-pre"}},
	{"v1.2.3+meta-pre.sha.256a", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: "meta-pre.sha.256a"}},
}

var extendedTests = []parseTestCase{
	{"v1.2.3.4.5", nil},
	{"1.2.3-aplphaあああ", nil},
	{"1.0.0", &Version{vPrefix: false, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "", build: ""}},
	{"1.2.3.4", &Version{vPrefix: false, core: Core{component: [4]uint16{1, 2, 3, 4}, length: 4}, prerelease: "", build: ""}},
	{"v1.2.3.4", &Version{vPrefix: true, core: Core{component: [4]uint16{1, 2, 3, 4}, length: 4}, prerelease: "", build: ""}},
	{"1.2.3.4-pre+meta", &Version{vPrefix: false, core: Core{component: [4]uint16{1, 2, 3, 4}, length: 4}, prerelease: "pre", build: "meta"}},
}

func concat[V any](seqs ...iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, seq := range seqs {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}

func TestParse(t *testing.T) {
	for tc := range concat(slices.Values(tests), slices.Values(extendedTests)) {
		ver, err := Parse(tc.in)
		if tc.out == nil {
			if err == nil {
				t.Errorf("%q: should be non-nil error but nil", tc.in)
			}
		} else {
			if err != nil {
				t.Errorf("%q: should be nil error but is %q", tc.in, err)
				continue
			}
			if *tc.out != ver {
				t.Errorf("%q: not equal:\nexpected = %#v\nactual = %#v", tc.in, *tc.out, ver)
			}
		}
	}
}

func TestVersion_With(t *testing.T) {
	var ver Version
	if ver.String() != "0.0.0" {
		t.Fatalf("incorrect String impl")
	}
	ver = ver.WithCore(MustNewCore([]uint16{1, 2, 3, 4}))
	if ver.String() != "1.2.3.4" {
		t.Fatalf("incorrect WithCore impl")
	}
	ver = ver.WithV(true)
	if ver.String() != "v1.2.3.4" {
		t.Fatalf("incorrect WithCore impl")
	}
	var err error
	ver, err = ver.WithPreRelease("alpha.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver.String() != "v1.2.3.4-alpha.3" {
		t.Fatalf("incorrect WithCore impl")
	}

	ver, err = ver.WithPreRelease("beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver.String() != "v1.2.3.4-beta" {
		t.Fatalf("incorrect WithCore impl")
	}

	ver, err = ver.WithBuild("meta-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver.String() != "v1.2.3.4-beta+meta-123" {
		t.Fatalf("incorrect WithCore impl")
	}
}

func TestVersion_String_MarshalText_UnmarshalText_MarshalJSON_UnmarshalJSON(t *testing.T) {
	for tc := range concat(slices.Values(tests), slices.Values(extendedTests)) {
		for _, arshalers := range []struct {
			unmarshal func(s string) (Version, error)
			marshal   func(v Version) ([]byte, error)
		}{
			{
				func(s string) (Version, error) {
					var v Version
					err := v.UnmarshalText([]byte(s))
					return v, err
				},
				Version.MarshalText,
			},
			{
				func(s string) (Version, error) {
					var v Version
					err := v.UnmarshalJSON([]byte(strconv.Quote(s)))
					return v, err
				},
				func(v Version) ([]byte, error) {
					bin, err := v.MarshalJSON()
					if err == nil {
						return bin[1 : len(bin)-1], nil
					}
					return bin, err
				},
			},
		} {
			ver, err := arshalers.unmarshal(tc.in)
			if tc.out == nil {
				if err == nil {
					t.Errorf("%q: should be non-nil error but nil", tc.in)
				}
				bin, err := arshalers.marshal(ver)
				if err != nil {
					t.Errorf("marshaling failed: %v", err)
				}
				if string(bin) != "0.0.0" {
					t.Errorf("zero value must marshaled to \"0.0.0\", but is %q", string(bin))
				}
			} else {
				if err != nil {
					t.Errorf("%q: should be nil error but is %q", tc.in, err)
					continue
				}
				if *tc.out != ver {
					t.Errorf("%q: not equal:\nexpected = %#v\nactual = %#v", tc.in, *tc.out, ver)
				}

				bin, err := arshalers.marshal(ver)
				if err != nil {
					t.Errorf("marshaling failed: %v", err)
				}
				if string(bin) != tc.in {
					t.Errorf("not equal:\nexpected = %s\nactual = %s", tc.in, string(bin))
				}
				if ver.String() != tc.in {
					t.Errorf("not equal:\nexpected = %s\nactual = %s", tc.in, ver.String())
				}
			}
		}
	}
}

func TestCompare(t *testing.T) {
	for i, ti := range tests {
		if ti.out == nil {
			continue
		}
		for j, tj := range tests {
			if tj.out == nil {
				continue
			}
			c := ti.out.Compare(*tj.out)
			var want int
			if ignoreMeta(*ti.out) == ignoreMeta(*tj.out) {
				want = 0
			} else if i < j {
				want = -1
			} else {
				want = +1
			}
			if c != want {
				t.Errorf("Compare(%q, %q) = %d, want %d", ti.in, tj.in, c, want)
			}
		}
	}
}

func Test_compare_by_sortable_string(t *testing.T) {
	for i, ti := range tests {
		if ti.out == nil {
			continue
		}

		if len(ti.out.PreReleaseSortable()) != 256 {
			t.Errorf("wrong leng: expected = 256, actual = %s", ti.out.PreReleaseSortable())
		}

		for j, tj := range tests {
			if tj.out == nil {
				continue
			}
			if ti.out.Core().Len() != tj.out.Core().Len() {
				continue
			}

			l := fmt.Sprintf("%016d_%s", ti.out.Core().Int64(), ti.out.PreReleaseSortable())
			r := fmt.Sprintf("%016d_%s", tj.out.Core().Int64(), tj.out.PreReleaseSortable())

			c := cmp.Compare(l, r)
			var want int
			if ignoreMeta(*ti.out) == ignoreMeta(*tj.out) {
				want = 0
			} else if i < j {
				want = -1
			} else {
				want = +1
			}
			if c != want {
				t.Errorf("Compare(%q, %q) = %d, want %d\nl = %s\nr = %s", ti.in, tj.in, c, want, l, r)
			}
		}
	}
}

func ignoreMeta(ver Version) Version {
	ver.build = ""
	return ver
}
