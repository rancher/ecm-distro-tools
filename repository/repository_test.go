package repository

import "testing"

func TestStripBackportTag(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{
			line: "[Release-1.24] Some backport",
			want: "Some backport",
		},
		{
			line: " [Release-1.24]  Some backport",
			want: "Some backport",
		},
		{
			line: "[Release 1.24] Some backport",
			want: "Some backport",
		},
		{
			line: "[release-1.24] Some backport",
			want: "Some backport",
		},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := stripBackportTag(tt.line); got != tt.want {
				t.Errorf("stripBackportTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
