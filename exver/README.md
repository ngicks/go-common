# exver

EXtended VERsion.

The extended [Sementic Versionning 2.0](https://semver.org/) that

- allows `v` prefix (like [go's semver](https://pkg.go.dev/golang.org/x/mod/semver) does), also permits versions without it.
- one extra version field is allowed (`a.b.c.d` is allowed)
  - That's rare, but there's software that works under this 4-fields versioning schema.
- shortened versions are also allowed (e.g. `v1`, `1.2`, etc)
  - These forms not can be suffixed with pre-release or build-meta.

## version upper limitation

Each segment of version fields are limited up to `9999`.

## numeric conversion

`Core.Int64` converts core parts (version without pre-release and build-meta) into numeric form so that it can be saved in any storage.

The conversion logic is roghly equivalent of `strconv.ParseInt(fmt.Sprintf("%04d%04d%04d%04d", a, b, c, d), 10, 64)` but more efficient.
