package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/ngicks/go-common/exver"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// versionFile is the single source file of the version package holding the
// top-level `const Version = "..."`.
type versionFile struct {
	path string
	fset *token.FileSet
	file *ast.File
}

// loadVersionFile loads <prefix>/<libver> and locates the file declaring
// `const Version`. Exactly one declaration is required across the package.
//
// The package is loaded with Dir set to the submodule directory so that a
// nested module resolves against its own go.mod instead of the root one.
func loadVersionFile(ctx context.Context, prefix, libver string) (*versionFile, error) {
	dir := "."
	if prefix != "" {
		dir = prefix
	}
	pth := path.Join(dir, libver)
	cfg := &packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax,
		Context: ctx,
		Dir:     filepath.FromSlash(dir),
	}
	pkgs, err := packages.Load(cfg, "./"+path.Clean(libver))
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", pth, err)
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected exactly one package at %s; got %d", pth, len(pkgs))
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		errs := make([]error, len(pkg.Errors))
		for i, e := range pkg.Errors {
			errs[i] = e
		}
		return nil, fmt.Errorf("loading %s: %w", pth, errors.Join(errs...))
	}

	var files []*ast.File
	for _, f := range pkg.Syntax {
		if len(versionLits(f)) > 0 {
			files = append(files, f)
		}
	}
	if len(files) != 1 || len(versionLits(files[0])) != 1 {
		return nil, fmt.Errorf(
			"expected exactly one top-level `const Version = \"...\"` in %s",
			pth,
		)
	}
	f := files[0]
	return &versionFile{
		path: pkg.Fset.Position(f.Pos()).Filename,
		fset: pkg.Fset,
		file: f,
	}, nil
}

// versionLits collects the string literals of top-level
// `const Version = "..."` declarations in f.
func versionLits(f *ast.File) []*ast.BasicLit {
	var lits []*ast.BasicLit
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name.Name != "Version" || i >= len(vs.Values) {
					continue
				}
				if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					lits = append(lits, lit)
				}
			}
		}
	}
	return lits
}

// current parses the version currently declared in the file.
func (v *versionFile) current() (exver.Version, error) {
	lit := versionLits(v.file)[0]
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return exver.Version{}, fmt.Errorf("unquoting Version %s in %s: %w", lit.Value, v.path, err)
	}
	ver, err := exver.Parse(s)
	if err != nil {
		return exver.Version{}, fmt.Errorf("parsing Version %q in %s: %w", s, v.path, err)
	}
	return ver, nil
}

// rewrite replaces the Version literal with ver and writes the formatted
// file back. Callable repeatedly on the same in-memory AST.
func (v *versionFile) rewrite(ver exver.Version) error {
	old := versionLits(v.file)[0]
	astutil.Apply(v.file, func(c *astutil.Cursor) bool {
		if c.Node() != old {
			return true
		}
		c.Replace(&ast.BasicLit{
			ValuePos: old.ValuePos,
			Kind:     token.STRING,
			Value:    strconv.Quote(ver.String()),
		})
		return false
	}, nil)

	var buf bytes.Buffer
	if err := format.Node(&buf, v.fset, v.file); err != nil {
		return fmt.Errorf("formatting %s: %w", v.path, err)
	}
	if err := os.WriteFile(v.path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", v.path, err)
	}
	return nil
}
