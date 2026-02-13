package input

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMouseButton(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected MouseButton
	}{
		{"left explicit", "left", ButtonLeft},
		{"right", "right", ButtonRight},
		{"middle", "middle", ButtonMiddle},
		{"empty defaults to left", "", ButtonLeft},
		{"unknown defaults to left", "invalid", ButtonLeft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseMouseButton(tt.input))
		})
	}
}
