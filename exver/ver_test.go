package exver

import (
	"iter"
	"slices"
	"testing"
)

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
