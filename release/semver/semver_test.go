package semver

import (
	"testing"

	"github.com/rancher/ecm-distro-tools/types"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version    string
		major      int
		minor      int
		patch      int
		prerelease string
		build      string
	}{
		{"v1.2.3", 1, 2, 3, "", ""},
		{"v2.0.0", 2, 0, 0, "", ""},
		{"v2.8", 2, 8, 0, "", ""},
		{"v1.28.2-rc1+rke2r1", 1, 28, 2, "-rc1", "rke2r1"},
		{"v0.1.0", 0, 1, 0, "", ""},
		{"v0.1.0", 0, 1, 0, "", ""},
	}

	for _, tt := range tests {
		v, err := ParseVersion(tt.version)
		if err != nil {
			t.Fatal(err)
		}
		if tt.major != v.Major {
			t.Fatalf("expected %d, got %d", tt.major, v.Major)
		}
		if tt.minor != v.Minor {
			t.Fatalf("expected %d, got %d", tt.minor, v.Minor)
		}
		// assert.NoError(t, err)
		// assert.Equal(t, tt.major, v.Major)
		// assert.Equal(t, tt.minor, v.Minor)
	}
}

func TestParsePattern(t *testing.T) {
	tests := []struct {
		pattern    string
		major      *int
		minor      *int
		patch      *int
		prerelease string
		build      string
	}{
		{"v1.0.1", types.IntPtr(1), types.IntPtr(0), types.IntPtr(1), "", ""},
		{"v2.1.3", types.IntPtr(2), types.IntPtr(1), types.IntPtr(3), "", ""},
		{"v0.0.0", types.IntPtr(0), types.IntPtr(0), types.IntPtr(0), "", ""},
		{"v0.0.0-rc1+abc", types.IntPtr(0), types.IntPtr(0), types.IntPtr(0), "-rc1", "+abc"},
		{"v1", types.IntPtr(1), nil, nil, "", ""},
		{"v0.x", types.IntPtr(0), nil, nil, "", ""},
		{"v2.8.x", types.IntPtr(2), types.IntPtr(8), nil, "", ""},
	}

	for _, tt := range tests {
		p, err := ParsePattern(tt.pattern)
		if err != nil {
			t.Fatal(err)
		}
		if p == nil {
			t.Fatal("pattern is nil")
		}

		if tt.major == nil {
			if p.Major != nil {
				t.Fatalf("expected nil, got %d", *p.Major)
			}
		} else {
			if p.Major == nil {
				t.Fatalf("expected %d, got nil", tt.major)
			}
			if *tt.major != *p.Major {
				t.Fatalf("expected %d, got %d", tt.major, p.Major)
			}
		}

		if tt.minor == nil {
			if p.Minor != nil {
				t.Fatalf("expected nil, got %d", *p.Minor)
			}
		} else {
			if p.Minor == nil {
				t.Fatalf("expected %d, got nil", tt.minor)
			}
			if *tt.minor != *p.Minor {
				t.Fatalf("expected %d, got %d", tt.minor, p.Minor)
			}
		}

		if tt.patch == nil {
			if p.Patch != nil {
				t.Fatalf("expected nil, got %d", *p.Patch)
			}
		} else {
			if p.Patch == nil {
				t.Fatalf("expected %d, got nil", tt.patch)
			}
			if *tt.patch != *p.Patch {
				t.Fatalf("expected %d, got %d", tt.patch, p.Patch)
			}
		}

		if tt.prerelease != p.Prerelease {
			t.Fatalf("expected %s, got %s", tt.prerelease, p.Prerelease)
		}
		if tt.build != p.Build {
			t.Fatalf("expected %s, got %s", tt.build, p.Build)
		}
	}
}

func TestPatternTest(t *testing.T) {
	tests := []struct {
		pattern  string
		version  string
		result   bool
		hasError bool
	}{
		{"v1.0.0", "v1.0.0", true, false},
		{"v1.0.0", "v1.0.1", false, false},
		{"v1.0.1", "v1.0.1", true, false},
		{"v1.0.1", "v1.0.2", false, false},
		{"v1.0.1", "v1.0.0", false, false},
		{"v1.0.1", "v1.0.1-rc1", true, false},
		{"v1.0.1-rc1", "v1.0.1-rc1", true, false},
		{"v1.0.1-rc1", "v1.0.1-rc2", false, false},
		{"v1.0.1+abc", "v1.0.1+abc", true, false},
		{"v1.0+abc", "v1.0.1+abc", true, false},
		{"v1.0.x+abc", "v1.0.1+abc", true, false},
		{"v1.0.1+abc", "v1.0.1+abc", true, false},
		{"v1.0.1-rc1+abc", "v1.0.1+abc", false, false},
		{"v1.0.1", "v1.0.1+abc", true, false},
		{"v2.8.x", "v2.8.1", true, false},
		{"v2.8.x", "v2.8.10-rc1", true, false},
	}

	for _, tt := range tests {
		pattern, err := ParsePattern(tt.pattern)
		if err != nil {
			t.Fatal(err)
		}
		if pattern == nil {
			t.Fatal("pattern is nil")
		}

		version, err := ParseVersion(tt.version)
		if err != nil {
			t.Fatal(err)
		}
		if version == nil {
			t.Fatal("version is nil")
		}

		result, err := pattern.Test(version)
		if !tt.hasError && err != nil {
			t.Fatal(err)

		}
		if tt.result != result {
			t.Fatalf("expected %t, got %t", tt.result, result)
		}
	}
}
