package main

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "0 minutes",
			duration: time.Second,
			expected: "0h0m",
		},
		{
			name:     "1 minute",
			duration: time.Minute,
			expected: "0h1m",
		},
		{
			name:     "2 minutes",
			duration: time.Minute * 2,
			expected: "0h2m",
		},
		{
			name:     "exactly 1 hour",
			duration: time.Minute * 60,
			expected: "1h0m",
		},
		{
			name:     "over 1 hour",
			duration: time.Minute * 61,
			expected: "1h1m",
		},
		{
			name:     "exactly 2 hours",
			duration: time.Minute * 120,
			expected: "2h0m",
		},
		{
			name:     "over 2 hours",
			duration: time.Minute * 121,
			expected: "2h1m",
		},
		{
			name:     "1 day and hours",
			duration: time.Hour * 25,
			expected: "1d1h",
		},
		{
			name:     "2 days exactly",
			duration: time.Hour * 48,
			expected: "2d",
		},

		{
			name:     "over 2 days",
			duration: time.Hour * 50,
			expected: "2d",
		},
		{
			name:     "363 days",
			duration: time.Hour * 24 * 363,
			expected: "363d",
		},
		{
			name:     "364 days",
			duration: time.Hour * 24 * 364,
			expected: "364d",
		},
		{
			name:     "365 days",
			duration: time.Hour * 24 * 365,
			expected: "1y",
		},
		{
			name:     "366 days",
			duration: time.Hour * 24 * 366,
			expected: "1y",
		},
		{
			name:     "2 years",
			duration: time.Hour * 24 * 365 * 2,
			expected: "2y",
		},
		{
			name:     "10 years",
			duration: time.Hour * 24 * 365 * 10,
			expected: "10y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := formatDuration(tt.duration)
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}
