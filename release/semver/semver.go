package semver

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rancher/ecm-distro-tools/types"

	"golang.org/x/mod/semver"
)

type Pattern struct {
	Pattern    string
	Major      *int
	Minor      *int
	Patch      *int
	Prerelease string
	Build      string
}

type Version struct {
	Version    string
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

func majorMinorPatch(version string) (string, string, string, error) {
	if !strings.HasPrefix(version, "v") {
		return "", "", "", errors.New("version must start with 'v'")
	}

	// Remove the "v" prefix
	version = strings.TrimPrefix(version, "v")

	// Remove anything following a "-" or a "+"
	version = strings.FieldsFunc(version, func(c rune) bool {
		return c == '-' || c == '+'
	})[0]

	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return "", "", "", errors.New("invalid version")
	}
	major := parts[0]
	var minor, patch string

	if len(parts) > 1 {
		minor = parts[1]
	}

	if len(parts) > 2 {
		patch = parts[2]
	}

	return major, minor, patch, nil
}
func ParsePattern(pattern string) (*Pattern, error) {
	prerelease := semver.Prerelease(pattern)
	build := semver.Build(pattern)
	major, minor, patch, err := majorMinorPatch(pattern)
	if err != nil {
		return nil, err
	}

	p := Pattern{Pattern: pattern}

	if major == "" || major == "x" {
		p.Major = nil
	} else {
		majorInt, err := strconv.Atoi(major)
		if err != nil {
			return nil, err
		}
		p.Major = types.IntPtr(majorInt)
	}

	if minor == "" || minor == "x" {
		p.Minor = nil
	} else {
		minorInt, err := strconv.Atoi(minor)
		if err != nil {
			return nil, err
		}
		p.Minor = types.IntPtr(minorInt)
	}

	if patch == "" || patch == "x" {
		p.Patch = nil
	} else {
		patchInt, err := strconv.Atoi(patch)
		if err != nil {
			return nil, err
		}
		p.Patch = types.IntPtr(patchInt)
	}

	if prerelease == "" || prerelease == "-x" {
		p.Prerelease = ""
	} else {
		p.Prerelease = prerelease
	}

	if build == "" || build == "-x" {
		p.Build = ""
	} else {
		p.Build = build
	}

	return &p, nil
}

func ParseVersion(version string) (*Version, error) {
	if !semver.IsValid(version) {
		return nil, errors.New("invalid version")
	}

	major := semver.Major(version)
	majorMinor := semver.MajorMinor(version)
	canonical := semver.Canonical(version)
	prerelease := semver.Prerelease(version)

	v := Version{Version: version}

	majorInt, err := strconv.Atoi(strings.TrimPrefix(major, "v"))
	if err != nil {
		return nil, err
	}
	v.Major = majorInt

	minor := strings.TrimPrefix(majorMinor, major+".")
	minorInt, err := strconv.Atoi(minor)
	if err != nil {
		return nil, err
	}
	v.Minor = minorInt

	patchAndPrerelease := strings.TrimPrefix(canonical, majorMinor+".")
	patch := strings.SplitN(patchAndPrerelease, "-", 2)[0]
	patchInt, err := strconv.Atoi(patch)
	if err != nil {
		return nil, err
	}
	v.Patch = patchInt

	v.Prerelease = prerelease
	v.Build = semver.Build(version)

	return &v, nil
}

// returns true if the semver pattern p is matched by version v
func (p *Pattern) Test(v *Version) (bool, error) {
	if p == nil {
		return false, errors.New("invalid pattern")
	}
	if v == nil {
		return false, errors.New("invalid version")
	}

	if p.Major != nil {
		if *p.Major != v.Major {
			return false, nil
		}
	}

	if p.Minor != nil {
		if *p.Minor != v.Minor {
			return false, nil
		}
	}

	if p.Patch != nil {
		if *p.Patch != v.Patch {
			return false, nil
		}
	}

	if p.Prerelease != "" && p.Prerelease != "x" {
		if p.Prerelease != v.Prerelease {
			return false, nil
		}
	}

	if p.Build != "" && p.Build != "x" {
		if p.Build != v.Build {
			return false, nil
		}
	}

	return true, nil
}
