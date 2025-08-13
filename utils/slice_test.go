package utils

import "testing"

func TestSliceContains(t *testing.T) {
	runSliceContains(t, []struct {
		name     string
		haystack []string
		needle   string
		expected bool
	}{
		{
			name:     "string contains",
			haystack: []string{"apple", "carrot", "egg"},
			needle:   "egg",
			expected: true,
		},
		{
			name:     "string does not contain",
			haystack: []string{"apple", "carrot", "egg"},
			needle:   "net",
			expected: false,
		},
	})
}

func runSliceContains[T comparable](t *testing.T,
	tests []struct {
		name     string
		haystack []T
		needle   T
		expected bool
	}) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SliceContains(tt.haystack, tt.needle)
			if got != tt.expected {
				t.Errorf("SliceContains(%v, %v) = %v; expected %v",
					tt.haystack, tt.needle, got, tt.expected)
			}
		})
	}
}
