package exver

import (
	"iter"
	"slices"
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
	{"v1.0.0-alpha", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "alpha", build: ""}},
	{"v1.0.0-alpha.1", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "alpha.1", build: ""}},
	{"v1.0.0-alpha.beta", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "alpha.beta", build: ""}},
	{"v1.0.0-beta", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "beta", build: ""}},
	{"v1.0.0-beta.2", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "beta.2", build: ""}},
	{"v1.0.0-beta.11", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "beta.11", build: ""}},
	{"v1.0.0-rc.1", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "rc.1", build: ""}},
	{"v1", &Version{v: true, core: Core{component: [4]uint16{1}, length: 1}, prerelease: "", build: ""}},
	{"v1.0", &Version{v: true, core: Core{component: [4]uint16{1}, length: 2}, prerelease: "", build: ""}},
	{"v1.0.0", &Version{v: true, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "", build: ""}},
	{"v1.2", &Version{v: true, core: Core{component: [4]uint16{1, 2}, length: 2}, prerelease: "", build: ""}},
	{"v1.2.0", &Version{v: true, core: Core{component: [4]uint16{1, 2}, length: 3}, prerelease: "", build: ""}},
	{"v1.2.3-456", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456", build: ""}},
	{"v1.2.3-456.789", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456.789", build: ""}},
	{"v1.2.3-456-789", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456-789", build: ""}},
	{"v1.2.3-456a", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "456a", build: ""}},
	{"v1.2.3-pre", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "pre", build: ""}},
	{"v1.2.3-pre+meta", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "pre", build: "meta"}},
	{"v1.2.3-pre.1", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "pre.1", build: ""}},
	{"v1.2.3-zzz", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "zzz", build: ""}},
	{"v1.2.3", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: ""}},
	{"v1.2.3+meta", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: "meta"}},
	{"v1.2.3+meta-pre", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: "meta-pre"}},
	{"v1.2.3+meta-pre.sha.256a", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3}, length: 3}, prerelease: "", build: "meta-pre.sha.256a"}},
}

var extendedTests = []parseTestCase{
	{"v1.2.3.4.5", nil},
	{"1.2.3-aplphaあああ", nil},
	{"1.0.0", &Version{v: false, core: Core{component: [4]uint16{1}, length: 3}, prerelease: "", build: ""}},
	{"1.2.3.4", &Version{v: false, core: Core{component: [4]uint16{1, 2, 3, 4}, length: 4}, prerelease: "", build: ""}},
	{"v1.2.3.4", &Version{v: true, core: Core{component: [4]uint16{1, 2, 3, 4}, length: 4}, prerelease: "", build: ""}},
	{"1.2.3.4-pre+meta", &Version{v: false, core: Core{component: [4]uint16{1, 2, 3, 4}, length: 4}, prerelease: "pre", build: "meta"}},
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

func TestCompare(t *testing.T) {
	for i, ti := range tests {
		if ti.out == nil {
			continue
		}
		for j, tj := range tests {
			if tj.out == nil {
				continue
			}
			cmp := ti.out.Compare(*tj.out)
			var want int
			if ignoreMeta(*ti.out) == ignoreMeta(*tj.out) {
				want = 0
			} else if i < j {
				want = -1
			} else {
				want = +1
			}
			if cmp != want {
				t.Errorf("Compare(%q, %q) = %d, want %d", ti.in, tj.in, cmp, want)
			}
		}
	}
}

func ignoreMeta(ver Version) Version {
	ver.build = ""
	return ver
}
