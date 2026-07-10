package prbuilder

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// ParseVersion extracts version from tag based on mapping type
// Examples:
//   - "major":       v10.3.2 -> "10"
//   - "major.minor": v10.3.2 -> "10.3"
func ParseVersion(tag, mappingType string) (string, error) {
	if !semver.IsValid(tag) {
		return "", fmt.Errorf("invalid tag format: %s (expected format: vX.Y.Z or X.Y.Z)", tag)
	}

	switch mappingType {
	case "major.minor":
		tag = semver.MajorMinor(tag)
	case "major", "":
		tag = semver.Major(tag)
	default:
		return "", fmt.Errorf("invalid version_mapping_type: %s", mappingType)
	}
	tag = strings.TrimPrefix(tag, "v")

	return tag, nil
}
