# tools

misc tools. Each executable is its own Go module so it can be run directly:

```sh
go run github.com/ngicks/go-common/tools/<name>@latest
```

- `golatestpatchver`: prints the latest Go patch version for the minor version in go.mod.
- `bump-libver`: cuts a release of a Go module that keeps its version in `<module-root>/internal/libver` — rewrites `const Version`, commits, tags, bumps to the next `-devel` version, and pushes.

If anything grows larger, internal logic may be separated into other go modules but executables remain same.
