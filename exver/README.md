# exver

Package exver implements parser and type for extended [Sementic Versionning 2.0](https://semver.org/).

exver is superset of [go's semver](https://pkg.go.dev/golang.org/x/mod/semver).

The general form of a semantic version string accepted by this package is

```
[v]MAJOR[.MINOR[.PATCH[.EXTRA][-PRERELEASE][+BUILD]]]
```

Basically same as go's semver package.
But small extensios are added:

- `v` prefix is optional.
- One EXTRA version component is added.
- All version components are limited at maximum of `9999`.
  - This limitation is placed to convert version to numeric form.

## numeric conversion

`Core.Int64` converts core parts (version without pre-release and build-meta) into numeric form
so that it can be saved in any storage and compared naturally within them.

The conversion logic is roghly equivalent of `strconv.ParseInt(fmt.Sprintf("%04d%04d%04d%04d", a, b, c, d), 10, 64)` but more efficient.
