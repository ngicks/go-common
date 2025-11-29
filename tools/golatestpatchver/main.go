package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/ngicks/go-common/exver"
	"github.com/ngicks/go-iterator-helper/hiter"
)

// curl https://proxy.golang.org/golang.org/toolchain/@v/list

var currentVersion = flag.String("ver", "", "current version. If empty, `go mod edit -json` will be used to retrieve go.mod version.")

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	curVer := *currentVersion
	if curVer == "" {
		var err error
		curVer, err = goVersionFromGoMod(ctx)
		if err != nil {
			panic(err)
		}
	}

	ver, err := parseGoVer(curVer)
	if err != nil {
		panic(err)
	}

	vers, err := listGoVersion(ctx)
	if err != nil {
		panic(err)
	}

	// bin, _ := json.MarshalIndent(vers, "", "    ")
	// fmt.Println(string(bin))

	found, ok := pickLatestSubVer(vers, ver.Ver)
	if !ok {
		panic("version not found: query=" + curVer)
	}
	fmt.Println(found.Org)
}

type verPair struct {
	Org string
	Ver exver.Version
}

func goVersionFromGoMod(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		"go",
		"mod", "edit", "-json",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}
	type goMod struct {
		Go string
	}
	var m goMod
	err = json.Unmarshal(bytes.TrimSpace(out), &m)
	if err != nil {
		return "", nil
	}
	return m.Go, nil
}

func parseGoVer(ver string) (parsed verPair, err error) {
	org := ver
	if strings.Count(ver, ".") <= 1 {
		var suf string
		ver, suf = splitSuf(ver)
		ver = ver + ".0" + suf
	}
	ver = strings.Replace(ver, "beta", "-beta.", 1)
	ver = strings.Replace(ver, "rc", "-rc.", 1)
	v, err := exver.Parse(ver)
	if err != nil {
		return
	}
	return verPair{org, v}, nil
}

func splitSuf(s string) (l, r string) {
	i := -1
	if strings.Contains(s, "beta") {
		i = strings.LastIndex(s, "beta")
	}
	if strings.Contains(s, "rc") {
		i = strings.LastIndex(s, "rc")
	}
	if i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}

func listGoVersion(ctx context.Context) ([]verPair, error) {
	cmd := exec.CommandContext(
		ctx,
		"git",
		"ls-remote", "--tags", "https://go.googlesource.com/go", "go*",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lis, err := hiter.TryCollect(
		hiter.Divide(
			parseGoVer,
			hiter.Map(
				func(s string) string {
					return strings.TrimSpace(s[strings.LastIndex(s, "refs/tags/go")+len("refs/tags/go"):])
				},
				hiter.Filter(
					func(s string) bool { return strings.LastIndex(s, "refs/tags/go") >= 0 },
					strings.SplitSeq(strings.TrimSpace(string(out)), "\n"),
				),
			),
		),
	)
	slices.SortFunc(
		lis,
		func(l, r verPair) int {
			return -l.Ver.Compare(r.Ver)
		},
	)
	return lis, err
}

func pickLatestSubVer(vers []verPair, target exver.Version) (verPair, bool) {
	v, i := hiter.FindFunc(
		func(ver verPair) bool {
			return ver.Ver.Core().Minor() == target.Core().Minor()
		},
		slices.Values(vers),
	)
	return v, i >= 0
}
