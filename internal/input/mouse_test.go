package input

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMoveMouse_ReachesDestination(t *testing.T) {
	// Move to a known position, then verify we arrived.
	// Requires Accessibility permission to post CGEvents.
	// Use a position well away from screen edges and menu bar to avoid clamping.
	target := struct{ x, y int }{400, 400}

	MoveMouse(target.x, target.y)
	time.Sleep(100 * time.Millisecond) // let CGEvent propagate

	gotX, gotY := GetMousePosition()
	assert.InDelta(t, target.x, gotX, 2, "mouse X should be near target")
	assert.InDelta(t, target.y, gotY, 2, "mouse Y should be near target")
}
