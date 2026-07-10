package prbuilder

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		tag         string
		mappingType string
		expected    string
		expectError bool
	}{
		// Major version mapping
		{tag: "v10.3.2", mappingType: "major", expected: "10", expectError: false},
		{tag: "v1.2.3", mappingType: "major", expected: "1", expectError: false},
		{tag: "v20.15.8", mappingType: "major", expected: "20", expectError: false},
		{tag: "10.3.2", mappingType: "major", expected: "10", expectError: false}, // No 'v' prefix

		// Major.minor version mapping
		{tag: "v10.3.2", mappingType: "major.minor", expected: "10.3", expectError: false},
		{tag: "v1.2.3", mappingType: "major.minor", expected: "1.2", expectError: false},
		{tag: "v20.15.8", mappingType: "major.minor", expected: "20.15", expectError: false},
		{tag: "10.3.2", mappingType: "major.minor", expected: "10.3", expectError: false}, // No 'v' prefix

		// Empty mapping type defaults to major
		{tag: "v10.3.2", mappingType: "", expected: "10", expectError: false},

		// Error cases
		{tag: "v1", mappingType: "major", expected: "", expectError: true},                     // Too few parts
		{tag: "v1.2", mappingType: "major", expected: "1", expectError: false},                 // Minimum valid
		{tag: "invalid", mappingType: "major", expected: "", expectError: true},                // No dots
		{tag: "v10.3.2", mappingType: "invalid-type", expected: "", expectError: true},         // Invalid mapping type
		{tag: "v10.3.2.4.5", mappingType: "major", expected: "10", expectError: false},         // Extra parts ignored
		{tag: "v10.3.2.4.5", mappingType: "major.minor", expected: "10.3", expectError: false}, // Extra parts ignored
	}

	for _, tt := range tests {
		t.Run(tt.tag+"_"+tt.mappingType, func(t *testing.T) {
			got, err := ParseVersion(tt.tag, tt.mappingType)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseVersion(%q, %q) expected error, got nil", tt.tag, tt.mappingType)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseVersion(%q, %q) unexpected error: %v", tt.tag, tt.mappingType, err)
				return
			}

			if got != tt.expected {
				t.Errorf("ParseVersion(%q, %q) = %q, want %q", tt.tag, tt.mappingType, got, tt.expected)
			}
		})
	}
}
