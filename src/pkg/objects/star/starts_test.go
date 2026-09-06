package star

import (
	"testing"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
)

// TestExplode pins the two properties the grid has to hold: the field carries
// exactly the requested number of stars, and every one of them is on the canvas.
// The grid used to be sized from the canvas area divided by the count, which
// overshot both axes, so part of the field was placed where it could never be
// drawn.
func TestExplode(t *testing.T) {
	canvas := config.CanvasBoundingBox()

	for _, count := range []int{0, 1, 7, 50, 200} {
		stars := Explode(count)

		if len(stars) != count {
			t.Errorf("Explode(%d): got %d stars, want %d", count, len(stars), count)
		}

		for _, star := range stars {
			if star.Position.X < 0 || star.Position.X.Float() > canvas.OriginalWidth ||
				star.Position.Y < 0 || star.Position.Y.Float() > canvas.OriginalHeight {

				t.Errorf("Explode(%d): star at %s is off the canvas", count, star.Position)
			}
		}
	}
}
