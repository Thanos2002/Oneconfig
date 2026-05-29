package runtime

import (
	"testing"
)

func TestMatchesVersion(t *testing.T) {
	tests := []struct {
		installed string
		required  string
		expected  bool
	}{
		{"v20.11.0", "20.x", true},
		{"v20.11.0", "20.11", true},
		{"v20.11.0", "18.x", false},
		{"Python 3.11.5", "3.11", true},
		{"Python 3.11.5", "3.12", false},
		{"v20.11.0", "20.11.0", true},
	}

	for _, tt := range tests {
		result := matchesVersion(tt.installed, tt.required)
		if result != tt.expected {
			t.Errorf("matchesVersion(%q, %q) = %v, want %v", tt.installed, tt.required, result, tt.expected)
		}
	}
}
