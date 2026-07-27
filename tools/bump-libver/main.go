// Command bump-libver cuts a release of a Go module that keeps its version
// string in <module-root>/internal/libver as a top-level
// `const Version = "..."` declaration.
//
// Usage:
//
//	go run github.com/ngicks/go-common/tools/bump-libver@latest [flags] [<submodule-dir>]
//
// Workflow:
//  1. Load <submodule-dir>/internal/libver (golang.org/x/tools/go/packages)
//     and locate the single top-level `const Version = "..."`.
//  2. Parse the current value with exver and compute the release version.
//     By default only the patch component moves: a "-devel" cycle already
//     carries the pre-bumped patch, so the release just strips the
//     pre-release. Pass -bump minor / -bump major for larger releases,
//     or -release for full control.
//  3. Rewrite the constant to the release version (astutil + go/format),
//     commit ("🔖: release <tag>"), tag.
//  4. Rewrite to the next development version (patch+1, "-devel"; override
//     with -next), commit ("👷: start <tag> development cycle").
//  5. Push the branch and the new tag to origin.
//
// Committing and pushing are opt-out: -no-push stops after the local
// commits and tag; -no-commit stops after rewriting the file to the
// release version (no tag, no next -devel bump, no dirty-tree check).
//
// <submodule-dir> is empty for the root module. For a Go submodule it is the
// module directory relative to the repository root (subpkg, nested/dir); the
// git tag is prefixed with it (subpkg/v0.2.0) while the bare version is
// written into the submodule's internal/libver.
//
// Run from the repository root, or point -C at it.
//
// Cross-platform by virtue of being a Go program: the same source compiles
// and runs on Linux, macOS, and Windows.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ngicks/go-common/exver"
)

var (
	flagChdir    = flag.String("C", "", "change to this directory (the repository root) before doing anything")
	flagBump     = flag.String("bump", "patch", "version component to bump: patch, minor or major")
	flagRelease  = flag.String("release", "", "explicit release version (overrides -bump); must NOT end in -devel")
	flagNext     = flag.String("next", "", "explicit next development version; must end in -devel")
	flagDryRun   = flag.Bool("dry-run", false, "print the computed versions and exit without changing anything")
	flagNoCommit = flag.Bool("no-commit", false, "only rewrite internal/libver to the release version; skip commit, tag, next -devel bump and push")
	flagNoPush   = flag.Bool("no-push", false, "commit and tag locally but do not push to origin")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(
			flag.CommandLine.Output(),
			`usage: bump-libver [flags] [<submodule-dir>]

  submodule-dir   directory of the Go submodule to release, relative to the
                  repository root (subpkg, nested/dir). Empty releases the
                  root module. The git tag is prefixed with it
                  (subpkg/v0.2.0); the version file is
                  <submodule-dir>/internal/libver.

The current version is read from internal/libver's top-level
const Version = "..." and incremented; by default only the patch component
moves (a -devel cycle already carries the pre-bumped patch, so the release
just strips the pre-release suffix).

flags:
`,
		)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if *flagChdir != "" {
		if err := os.Chdir(*flagChdir); err != nil {
			fmt.Fprintln(os.Stderr, "bump-libver:", err)
			os.Exit(1)
		}
	}
	opts := options{
		bump:     *flagBump,
		release:  *flagRelease,
		next:     *flagNext,
		dryRun:   *flagDryRun,
		noCommit: *flagNoCommit,
		noPush:   *flagNoPush,
	}
	if err := run(ctx, flag.Arg(0), opts); err != nil {
		fmt.Fprintln(os.Stderr, "bump-libver:", err)
		os.Exit(1)
	}
}

type options struct {
	bump, release, next      string
	dryRun, noCommit, noPush bool
}

func run(ctx context.Context, prefix string, opts options) error {
	if err := validatePrefix(prefix); err != nil {
		return err
	}

	vf, err := loadVersionFile(ctx, prefix)
	if err != nil {
		return err
	}

	cur, err := vf.current()
	if err != nil {
		return err
	}

	release, err := releaseVersion(cur, opts.bump, opts.release)
	if err != nil {
		return err
	}
	next, err := nextDevVersion(release, opts.next)
	if err != nil {
		return err
	}

	tag := withPrefix(prefix, release)
	nextTag := withPrefix(prefix, next)

	if opts.dryRun {
		fmt.Printf("current:  %s (%s)\nrelease:  %s\nnext-dev: %s\n", cur, vf.path, tag, nextTag)
		return nil
	}

	if opts.noCommit {
		if err := vf.rewrite(release); err != nil {
			return err
		}
		fmt.Printf("rewrote %s to %s; commit, tag and push skipped (-no-commit).\n", vf.path, release)
		return nil
	}

	if err := requireCleanWorkTree(ctx); err != nil {
		return err
	}
	if err := requireTagAbsent(ctx, tag); err != nil {
		return err
	}

	if err := vf.rewrite(release); err != nil {
		return err
	}
	if err := git(ctx, "add", "--", vf.path); err != nil {
		return err
	}
	if err := git(ctx, "commit", "-m", "🔖: release "+tag); err != nil {
		return err
	}
	if err := git(ctx, "tag", "-a", tag, "-m", tag); err != nil {
		return err
	}

	if err := vf.rewrite(next); err != nil {
		return err
	}
	if err := git(ctx, "add", "--", vf.path); err != nil {
		return err
	}
	if err := git(ctx, "commit", "-m", "👷: start "+nextTag+" development cycle"); err != nil {
		return err
	}

	if opts.noPush {
		fmt.Printf(
			"released %s; bumped to %s; push skipped (-no-push): run `git push && git push origin %s`.\n",
			tag,
			nextTag,
			tag,
		)
		return nil
	}

	if err := git(ctx, "push"); err != nil {
		return err
	}
	if err := git(ctx, "push", "origin", tag); err != nil {
		return err
	}

	fmt.Printf(
		"released %s; bumped to %s; pushed branch and tag %s to origin.\n",
		tag,
		nextTag,
		tag,
	)
	return nil
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return nil
	}
	for _, c := range strings.Split(prefix, "/") {
		if c == "" || c == "." || c == ".." || strings.ContainsAny(c, `\:`) {
			return fmt.Errorf("submodule-dir must be a clean slash-separated relative path; got %q", prefix)
		}
	}
	return nil
}

func withPrefix(prefix string, v exver.Version) string {
	if prefix == "" {
		return v.String()
	}
	return prefix + "/" + v.String()
}
