//go:build darwin

package capture

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveDisplayBounds(t *testing.T) {
	displays := ActiveDisplayBounds()
	require.NotEmpty(t, displays, "should find at least one active display")

	for i, d := range displays {
		assert.Positive(t, d.Width, "display %d width must be positive", i)
		assert.Positive(t, d.Height, "display %d height must be positive", i)
	}

	// Primary display should have origin (0,0)
	assert.Equal(t, 0, displays[0].X, "primary display X should be 0")
	assert.Equal(t, 0, displays[0].Y, "primary display Y should be 0")
}
