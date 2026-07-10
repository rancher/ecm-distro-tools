package prbuilder

import (
	"fmt"
	"strings"
)

// ParseVersion extracts version from tag based on mapping type
// Examples:
//   - "major":       v10.3.2 -> "10"
//   - "major.minor": v10.3.2 -> "10.3"
func ParseVersion(tag, mappingType string) (string, error) {
	// Remove 'v' prefix if present
	tag = strings.TrimPrefix(tag, "v")

	// Split by dots
	parts := strings.Split(tag, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid tag format: %s (expected format: vX.Y.Z or X.Y.Z)", tag)
	}

	switch mappingType {
	case "major.minor":
		return parts[0] + "." + parts[1], nil
	case "major", "":
		return parts[0], nil
	default:
		return "", fmt.Errorf("invalid version_mapping_type: %s", mappingType)
	}
}
