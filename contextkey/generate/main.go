package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	i           = flag.String("i", "template.json", "template for generated types.")
	o           = flag.String("o", "./", "output dir")
	keyTypeFunc = flag.String("key-type-func", "", "func name which takes type name as string and returns key type for the value.")
	pkgName     = flag.String("pkg-name", "", "package name for generated files")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"\tgenerates context keys and convenient setter/getter functions for the key.\n"+
				"\tinput `template` file must be json file directly unmarshalable to slice of the type\n"+
				"\n"+
				"\ttype rawInfo struct {\n"+
				"\t\tFileName       string   // Name of generated file will be this value suffixed with \".generated.go\" and \".generated_test.go\" \n"+
				"\t\tName           string   // Name used for functions and context key variable.\n"+
				"\t\tType           string   // Type of associated value for key.\n"+
				"\t\tDefaultExpr    string   // Expr of default value for Name. e.g. slog.Default().\n"+
				"\t\tNonDefaultExpr string   // Only used in test. Expr of non default value for Name. Set if fallen back value is non deterministic, e.g. calling new(T).\n"+
				"\t\tEqualFunc      string   // Only used in test. Function tests equality of 2 any value. default: func (v1, v2 any) bool { return v1 == v2 }.\n"+
				"\t\tImports        []string // Imports for generated code. Generator adds \"context\", encloses imports with double quotation, and prefix each line with a tab letter.\n"+
				"\t\tEmitTest       bool     // Whether the generator emits test for the name.\n"+
				"\t}\n"+
				"\n",
		)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	f := must(os.Open(*i))
	var tt []rawInfo
	must1(json.NewDecoder(f).Decode(&tt))
	for _, t := range tt {
		out := must(os.Create(filepath.Join(*o, t.FileName+".generated.go")))
		must1(fileTemplate.Execute(out, t.Into(*pkgName, *keyTypeFunc)))
		if t.EmitTest {
			outTest := must(os.Create(filepath.Join(*o, t.FileName+".generated_test.go")))
			must1(testTemplate.Execute(outTest, t.Into(*pkgName, *keyTypeFunc)))
		}
	}
}

func must1(err error) {
	if err != nil {
		panic(err)
	}
}

func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}

type rawInfo struct {
	FileName       string
	Name           string
	Type           string
	DefaultExpr    string
	NonDefaultExpr string
	EqualFunc      string
	Imports        []string
	EmitTest       bool
}

func (i rawInfo) Into(pkgName string, keyTypeFunc string) info {
	imports := make([]string, len(i.Imports))
	for j, imp := range i.Imports {
		if imp != "" {
			imp = "\t\"" + imp + "\""
		}
		// leave empty lines `as is``.
		imports[j] = imp
	}
	eq := i.EqualFunc
	if eq == "" {
		eq = "func (v1, v2 any) bool { return v1 == v2 }"
	}
	return info{
		PackageName:    pkgName,
		FileName:       i.FileName,
		Name:           i.Name,
		Type:           i.Type,
		DefaultExpr:    i.DefaultExpr,
		NonDefaultExpr: i.NonDefaultExpr,
		EqualFunc:      eq,
		Imports:        strings.Join(imports, "\n"),
		KeyTypeFunc:    keyTypeFunc,
	}
}

type info struct {
	PackageName    string
	FileName       string
	Name           string
	Type           string
	DefaultExpr    string
	NonDefaultExpr string
	EqualFunc      string
	Imports        string
	KeyTypeFunc    string
}

var fileTemplate = template.Must(template.New("t").Parse(`// Code generated by github.com/ngicks/go-common/contextkey/generate DO NOT EDIT
package {{.PackageName}}

import (
	"context"
{{.Imports}}
)

var (
	Key{{.Name}} = {{.KeyTypeFunc}}("{{.Type}}")
)

// With{{.Name}} returns a copy of parent in which the value associated with Key{{.Name}} is v.
// v associated with contexts for Key{{.Name}} can later be retrieved by any of Value method,
// Value{{.Name}}, Value{{.Name}}Fallback or Value{{.Name}}Default.
func With{{.Name}}(ctx context.Context, v {{.Type}}) context.Context {
	return context.WithValue(ctx, Key{{.Name}}, v)
}

// Value{{.Name}} returns {{.Type}} associated with ctx for Key{{.Name}}.
// ok is false if the value was not associated or other than {{.Type}}.
func Value{{.Name}}(ctx context.Context) (v {{.Type}}, ok bool) {
	val := ctx.Value(Key{{.Name}})
	if v, ok := val.({{.Type}}); ok {
		return v, true
	}
	var zero {{.Type}}
	return zero, false
}

// Value{{.Name}}Fallback returns {{.Type}} associated with ctx for Key{{.Name}},
// or in case the value was not associated to ctx, returns fallback.
func Value{{.Name}}Fallback(ctx context.Context, fallback {{.Type}}) {{.Type}} {
	v, ok := Value{{.Name}}(ctx)
	if ok {
		return v
	}
	return fallback
}

// Value{{.Name}}FallbackFn returns {{.Type}} associated with ctx for Key{{.Name}},
// or in case the value was not associated to ctx, returns calling result of fallbackFn.
func Value{{.Name}}FallbackFn(ctx context.Context, fallbackFn func() {{.Type}}) {{.Type}} {
	v, ok := Value{{.Name}}(ctx)
	if ok {
		return v
	}
	return fallbackFn()
}

// Value{{.Name}}Default returns {{.Type}} associated with ctx for Key{{.Name}}.
// In case the value was not associated, returns default value for the type.
//
// The default value is an evaluation result of {{.DefaultExpr}}.
func Value{{.Name}}Default(ctx context.Context) {{.Type}} {
	return Value{{.Name}}FallbackFn(ctx, func () {{.Type}} { return {{.DefaultExpr}} })
}
`))

var testTemplate = template.Must(template.New("t").Parse(`// Code generated by github.com/ngicks/go-common/contextkey/generate DO NOT EDIT
package {{.PackageName}}

import (
	"context"
	"testing"
{{.Imports}}
)

func Test{{.Name}}(t *testing.T) {
	var zero {{.Type}}

	ctx := context.Background()

	if v := ctx.Value(Key{{.Name}}); v != nil {
		t.Fatalf("there is value for key %v", Key{{.Name}})
	}

	noDefaultValue := {{.NonDefaultExpr}} 
	ctx = With{{.Name}}(ctx, noDefaultValue)

	v1 := ctx.Value(Key{{.Name}})
	if v1 == nil {
		t.Fatalf("no value for key %v", Key{{.Name}})
	}
	if _, ok := v1.({{.Type}}); !ok {
		t.Fatalf("different type associated, want = %s, got = %T", "{{.Type}}", v1)
	}

	getAndCompare := func(eq bool) func(v2 {{.Type}}, ok bool) {
		return func(v2 {{.Type}}, ok bool) {
			t.Helper()
			not := "not "
			if eq {
				not = ""
			}
			if ok != eq {
				t.Fatalf("different type associated, want = %s%s, got = %T", not, "{{.Type}}", v1)
			}

			if {{.EqualFunc}}(v1, v2) != eq {
				t.Fatalf("wrong value retrieved, want = %s%v, got = %v", not, v1, v2)
			}
		}
	}

	getAndCompare(true)(Value{{.Name}}(ctx))
	getAndCompare(true)(Value{{.Name}}Fallback(ctx, zero), true)
	getAndCompare(true)(Value{{.Name}}FallbackFn(ctx, func() {{.Type}} { return zero }), true)
	getAndCompare(true)(Value{{.Name}}Default(ctx), true)

	ctx = context.WithValue(ctx, Key{{.Name}}, struct{}{})
	getAndCompare(false)(Value{{.Name}}(ctx))
	getAndCompare(false)(Value{{.Name}}Fallback(ctx, zero), false)
	getAndCompare(false)(Value{{.Name}}FallbackFn(ctx, func() {{.Type}} { return zero }), false)
	getAndCompare(false)(Value{{.Name}}Default(ctx), false)

	ctx = context.Background()
	getAndCompare(false)(Value{{.Name}}(ctx))
	getAndCompare(false)(Value{{.Name}}Fallback(ctx, zero), false)
	getAndCompare(false)(Value{{.Name}}FallbackFn(ctx, func() {{.Type}} { return zero }), false)
	getAndCompare(false)(Value{{.Name}}Default(ctx), false)

	if v := Value{{.Name}}Fallback(ctx, noDefaultValue); !{{.EqualFunc}}(v, noDefaultValue) {
		t.Fatalf("Value{{.Name}}Fallback did not fall back correctly. want = %v, got = %v", noDefaultValue, v)
	}
	if v := Value{{.Name}}FallbackFn(ctx, func() {{.Type}} { return noDefaultValue }); !{{.EqualFunc}}(v, noDefaultValue) {
		t.Fatalf("Value{{.Name}}FallbackFn did not fall back correctly. want = %v, got = %v", noDefaultValue, v)
	}

	v := Value{{.Name}}Default(ctx)
	if !{{.EqualFunc}}(v, {{.DefaultExpr}}) {
		def1, def2 := {{.DefaultExpr}}, {{.DefaultExpr}}
		if  !{{.EqualFunc}}(def1, def2) {
			t.Logf("default value is non-deterministic; You must implement your own test for Value{{.Name}}Default(ctx)")
		} else {
			t.Fatalf("Value{{.Name}}Default did not fall back to default value")
		}
	}
}`))