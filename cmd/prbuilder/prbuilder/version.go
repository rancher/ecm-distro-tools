package prbuilder

import (
	"errors"
	"strings"

	"golang.org/x/mod/semver"
)

// ParseVersion extracts version from tag based on mapping type
// Examples:
//   - "major":       v10.3.2 -> "10"
//   - "major.minor": v10.3.2 -> "10.3"
func ParseVersion(tag, mappingType string) (string, error) {
	if !semver.IsValid(tag) {
		return "", errors.New("invalid tag format: " + tag + " (expected format: vX.Y.Z or X.Y.Z)")
	}

	switch mappingType {
	case "major.minor":
		tag = semver.MajorMinor(tag)
	case "major", "":
		tag = semver.Major(tag)
	default:
		return "", errors.New("invalid version_mapping_type: " + mappingType)
	}
	tag = strings.TrimPrefix(tag, "v")

	return tag, nil
}
