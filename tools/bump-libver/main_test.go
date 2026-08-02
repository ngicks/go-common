package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngicks/go-common/exver"
)

func TestIsDevel(t *testing.T) {
	cases := []struct {
		v    string
		want bool
	}{
		{"v0.1.0-devel", true},
		{"v0.1.0-devel.20240101", true},
		{"v0.1.0", false},
		{"v0.1.0-rc1", false},
		{"v0.1.0-alpha-devel", true},
	}
	for _, c := range cases {
		if got := isDevel(exver.MustParse(c.v)); got != c.want {
			t.Errorf("isDevel(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestReleaseVersion(t *testing.T) {
	cases := []struct {
		cur      string
		bump     string
		explicit string
		want     string
		wantErr  bool
	}{
		// a -devel cycle already carries the pre-bumped patch
		{cur: "v0.2.1-devel", bump: "patch", want: "v0.2.1"},
		{cur: "v0.2.1-devel.20240101", bump: "patch", want: "v0.2.1"},
		{cur: "v0.2.1-rc1", bump: "patch", want: "v0.2.1"},
		// no pre-release: the patch actually moves
		{cur: "v0.2.1", bump: "patch", want: "v0.2.2"},
		{cur: "v0.2.1-devel", bump: "minor", want: "v0.3.0"},
		{cur: "v0.2.1-devel", bump: "major", want: "v1.0.0"},
		{cur: "v1.2.3", bump: "major", want: "v2.0.0"},
		// short cores normalize before bumping
		{cur: "v1.2", bump: "patch", want: "v1.2.1"},
		{cur: "v0.2.1-devel", bump: "nope", wantErr: true},
		{cur: "v0.2.9999", bump: "patch", wantErr: true},
		// explicit overrides -bump entirely
		{cur: "v0.2.1-devel", bump: "patch", explicit: "v9.9.9", want: "v9.9.9"},
		{cur: "v0.2.1-devel", bump: "patch", explicit: "9.9.9-rc1", want: "v9.9.9-rc1"},
		{cur: "v0.2.1-devel", bump: "patch", explicit: "v9.9.9-devel", wantErr: true},
		{cur: "v0.2.1-devel", bump: "patch", explicit: "v9.9", wantErr: true},
		{cur: "v0.2.1-devel", bump: "patch", explicit: "bogus", wantErr: true},
	}
	for _, c := range cases {
		got, err := releaseVersion(exver.MustParse(c.cur), c.bump, c.explicit)
		if (err != nil) != c.wantErr {
			t.Errorf("releaseVersion(%q, %q, %q) err = %v, wantErr = %v", c.cur, c.bump, c.explicit, err, c.wantErr)
			continue
		}
		if !c.wantErr && got.String() != c.want {
			t.Errorf("releaseVersion(%q, %q, %q) = %q, want %q", c.cur, c.bump, c.explicit, got, c.want)
		}
	}
}

func TestNextDevVersion(t *testing.T) {
	cases := []struct {
		release  string
		explicit string
		want     string
		wantErr  bool
	}{
		{release: "v0.2.1", want: "v0.2.2-devel"},
		{release: "v1.2.99", want: "v1.2.100-devel"},
		{release: "v0.2.1-rc1", want: "v0.2.2-devel"},
		{release: "v0.2.9999", wantErr: true},
		{release: "v0.2.1", explicit: "v0.3.0-devel", want: "v0.3.0-devel"},
		{release: "v0.2.1", explicit: "0.3.0-devel", want: "v0.3.0-devel"},
		{release: "v0.2.1", explicit: "v0.3.0", wantErr: true},
		{release: "v0.2.1", explicit: "v0.3-devel", wantErr: true},
	}
	for _, c := range cases {
		got, err := nextDevVersion(exver.MustParse(c.release), c.explicit)
		if (err != nil) != c.wantErr {
			t.Errorf("nextDevVersion(%q, %q) err = %v, wantErr = %v", c.release, c.explicit, err, c.wantErr)
			continue
		}
		if !c.wantErr && got.String() != c.want {
			t.Errorf("nextDevVersion(%q, %q) = %q, want %q", c.release, c.explicit, got, c.want)
		}
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const libverSrc = `// Package libver holds the release-controlled version string.
package libver

// Version is the human-readable version string, rewritten by the bump-libver
// tool when cutting a release.
const Version = "v0.1.1-devel"
`

func TestLoadVersionFile_rewrite(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/x\n\ngo 1.25.0\n")
	writeFile(t, dir, "internal/libver/version.go", libverSrc)
	t.Chdir(dir)

	vf, err := loadVersionFile(t.Context(), "", "internal/libver")
	if err != nil {
		t.Fatalf("loadVersionFile: %v", err)
	}
	if want := filepath.Join(dir, "internal", "libver", "version.go"); vf.path != want {
		t.Errorf("path = %q, want %q", vf.path, want)
	}

	cur, err := vf.current()
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if cur.String() != "v0.1.1-devel" {
		t.Errorf("current = %q, want v0.1.1-devel", cur)
	}

	// release rewrite, then next-dev rewrite on the same in-memory AST
	for _, ver := range []string{"v0.1.1", "v0.1.2-devel"} {
		if err := vf.rewrite(exver.MustParse(ver)); err != nil {
			t.Fatalf("rewrite(%q): %v", ver, err)
		}
		got, err := os.ReadFile(vf.path)
		if err != nil {
			t.Fatal(err)
		}
		want := strings.Replace(libverSrc, `"v0.1.1-devel"`, quoted(ver), 1)
		if string(got) != want {
			t.Errorf("rewrite(%q) produced:\n%s\nwant:\n%s", ver, got, want)
		}
	}
}

func quoted(s string) string { return `"` + s + `"` }

func TestLoadVersionFile_submodule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/x\n\ngo 1.25.0\n")
	writeFile(t, dir, "sub/go.mod", "module example.com/x/sub\n\ngo 1.25.0\n")
	writeFile(t, dir, "sub/internal/libver/version.go", libverSrc)
	t.Chdir(dir)

	vf, err := loadVersionFile(t.Context(), "sub", "internal/libver")
	if err != nil {
		t.Fatalf("loadVersionFile: %v", err)
	}
	if want := filepath.Join(dir, "sub", "internal", "libver", "version.go"); vf.path != want {
		t.Errorf("path = %q, want %q", vf.path, want)
	}
}

func TestLoadVersionFile_customLibverDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/x\n\ngo 1.25.0\n")
	writeFile(t, dir, "meta/ver/version.go", libverSrc)
	t.Chdir(dir)

	vf, err := loadVersionFile(t.Context(), "", "meta/ver")
	if err != nil {
		t.Fatalf("loadVersionFile: %v", err)
	}
	if want := filepath.Join(dir, "meta", "ver", "version.go"); vf.path != want {
		t.Errorf("path = %q, want %q", vf.path, want)
	}
}

func TestLoadVersionFile_errors(t *testing.T) {
	t.Run("missing package", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/x\n\ngo 1.25.0\n")
		t.Chdir(dir)
		if _, err := loadVersionFile(t.Context(), "", "internal/libver"); err == nil {
			t.Fatal("expected error when internal/libver does not exist")
		}
	})

	t.Run("no Version const", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/x\n\ngo 1.25.0\n")
		writeFile(t, dir, "internal/libver/version.go", "package libver\n")
		t.Chdir(dir)
		if _, err := loadVersionFile(t.Context(), "", "internal/libver"); err == nil {
			t.Fatal("expected error when const Version is missing")
		}
	})

	t.Run("duplicate Version consts", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/x\n\ngo 1.25.0\n")
		writeFile(t, dir, "internal/libver/version.go", "package libver\n\nconst Version = \"v0.1.0\"\n")
		writeFile(t, dir, "internal/libver/other.go", "package libver\n\nconst Version2 = Version // not a dup\n")
		writeFile(t, dir, "internal/libver/dup.go", "package libver\n\nconst version = \"x\"\n")
		t.Chdir(dir)
		if _, err := loadVersionFile(t.Context(), "", "internal/libver"); err != nil {
			t.Fatalf("similarly named consts must not confuse detection: %v", err)
		}

		writeFile(t, dir, "internal/libver/dup2.go", "package libver\n\n//go:generate true\nconst Version = \"v0.2.0\"\n")
		if _, err := loadVersionFile(t.Context(), "", "internal/libver"); err == nil {
			t.Fatal("expected error on two Version declarations")
		}
	})
}
