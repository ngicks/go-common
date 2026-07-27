package main

import (
	"fmt"
	"strings"

	"github.com/ngicks/go-common/exver"
)

// isDevel reports whether v carries a -devel suffix (with or without an
// extra dotted qualifier such as -devel.20240101).
func isDevel(v exver.Version) bool {
	s := v.String()
	return strings.HasSuffix(s, "-devel") || strings.Contains(s, "-devel.")
}

// releaseVersion computes the version to release from the current
// internal/libver value.
//
// A "-devel" (or any pre-release) cycle already carries the pre-bumped
// patch, so bump == "patch" only strips the pre-release; the patch moves
// only when cur has no pre-release at all. minor / major zero the lower
// components as usual.
func releaseVersion(cur exver.Version, bump, explicit string) (exver.Version, error) {
	if explicit != "" {
		v, err := exver.Parse(explicit)
		if err != nil {
			return exver.Version{}, fmt.Errorf("parsing -release %q: %w", explicit, err)
		}
		if v.Core().Len() < 3 {
			return exver.Version{}, fmt.Errorf("-release must be full vMAJOR.MINOR.PATCH; got %q", explicit)
		}
		if isDevel(v) {
			return exver.Version{}, fmt.Errorf("-release must not end in -devel; got %q", explicit)
		}
		return v.WithV(true), nil
	}

	comp := cur.Core().Normalize().Component()
	major, minor, patch := comp[0], comp[1], comp[2]
	switch bump {
	case "patch":
		if cur.PreRelease() == "" {
			patch++
		}
	case "minor":
		minor, patch = minor+1, 0
	case "major":
		major, minor, patch = major+1, 0, 0
	default:
		return exver.Version{}, fmt.Errorf("-bump must be patch, minor or major; got %q", bump)
	}

	core, err := exver.NewCore([]uint16{major, minor, patch})
	if err != nil {
		return exver.Version{}, fmt.Errorf("bumping %q: %w", cur, err)
	}
	return exver.Version{}.WithV(true).WithCore(core), nil
}

// nextDevVersion computes the version the next development cycle starts at:
// the release with patch+1 and a "-devel" pre-release.
func nextDevVersion(release exver.Version, explicit string) (exver.Version, error) {
	if explicit != "" {
		v, err := exver.Parse(explicit)
		if err != nil {
			return exver.Version{}, fmt.Errorf("parsing -next %q: %w", explicit, err)
		}
		if v.Core().Len() < 3 {
			return exver.Version{}, fmt.Errorf("-next must be full vMAJOR.MINOR.PATCH-devel; got %q", explicit)
		}
		if !isDevel(v) {
			return exver.Version{}, fmt.Errorf("-next must end in -devel; got %q", explicit)
		}
		return v.WithV(true), nil
	}

	comp := release.Core().Normalize().Component()
	core, err := exver.NewCore([]uint16{comp[0], comp[1], comp[2] + 1})
	if err != nil {
		return exver.Version{}, fmt.Errorf("bumping %q: %w", release, err)
	}
	v, err := exver.Version{}.WithV(true).WithCore(core).WithPreRelease("devel")
	if err != nil {
		return exver.Version{}, err
	}
	return v, nil
}
