package release

import "testing"

func TestMajMin(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{
			version: "v3.24.1",
			want:    "v3.24",
		},
		{
			version: "v1.1.1-k3s1",
			want:    "v1.1",
		},
		{
			version: "1.2.3",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got, _ := majMin(tt.version); got != tt.want {
				t.Errorf("MajMin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimPeriods(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{
			version: "v3.24.1",
			want:    "v3241",
		},
		{
			version: "1.23.4",
			want:    "1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := trimPeriods(tt.version); got != tt.want {
				t.Errorf("trimPeriods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{
			version: "HELLO WORLD",
			want:    "HELLO WORLD",
		},
		{
			version: "hello world",
			want:    "Hello world",
		},
		{
			version: " hello world",
			want:    " Hello world",
		},
		{
			version: " [hello] world",
			want:    " [Hello] world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := capitalize(tt.version); got != tt.want {
				t.Errorf("capitalize() = %v, want %v", got, tt.want)
			}
		})
	}
}
