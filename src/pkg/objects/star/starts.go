package star

import (
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

type Stars []Star

// Twinkling reports whether any star still has to be painted.
// The starfield never moves, so once every star has been exhausted the
// offscreen canvas already holds the finished image and can simply be reused.
func (stars Stars) Twinkling() bool {
	for i := range stars {
		if !stars[i].Exhausted {
			return true
		}
	}

	return false
}

// Explode is a function that creates a number of stars.
// It creates a grid of cells and places stars in random positions within these cells.
// The number of stars is determined by the input parameter.
func Explode(num int) Stars {
	// Define the grid size
	canvasDimensions := config.CanvasBoundingBox()
	gridSize := (numeric.Number(canvasDimensions.OriginalWidth*canvasDimensions.OriginalHeight) / numeric.Number(num)).Root()
	newBox := numeric.Locate(canvasDimensions.OriginalWidth, canvasDimensions.OriginalHeight).Div(gridSize).ToBox()

	// Create a grid of cells and place stars in random positions within these cells
	var stars Stars
	for row := numeric.Number(0); row < newBox.Height; row++ {
		for col := numeric.Number(0); col < newBox.Width; col++ {
			if num <= 0 {
				return stars
			}

			stars = append(stars, *Twinkle(numeric.Locate(
				numeric.RandomRange(col, col+1),
				numeric.RandomRange(row, row+1),
			).Mul(numeric.Number(gridSize))))

			num--
		}
	}

	return stars
}
