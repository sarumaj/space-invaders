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

// Explode creates the given number of stars, one to a cell of a grid laid over
// the canvas, each placed at random within its cell.
//
// The grid used to be built from a square cell of the canvas area divided by the
// star count, which overshot the canvas on both axes: cells ran past the right
// and bottom edges, so a share of the stars were placed outside the drawable
// area and never seen, and the fill stopped part-way through the grid, leaving
// the bottom of the sky emptier than the top. Sizing the grid to the count
// instead keeps every star on the canvas and the coverage even.
func Explode(num int) Stars {
	if num <= 0 {
		return nil
	}

	canvasDimensions := config.CanvasBoundingBox()
	width := numeric.Number(canvasDimensions.OriginalWidth)
	height := numeric.Number(canvasDimensions.OriginalHeight)

	// Choose the column count that makes the cells as square as the canvas
	// allows, then take as many rows as the count needs.
	columns := (numeric.Number(num) * width / height).Root().Int()
	if columns < 1 {
		columns = 1
	}

	rows := (num + columns - 1) / columns
	cell := numeric.Locate(width/numeric.Number(columns), height/numeric.Number(rows))

	stars := make(Stars, 0, num)
	for i := 0; i < num; i++ {
		col, row := numeric.Number(i%columns), numeric.Number(i/columns)

		stars = append(stars, *Twinkle(numeric.Locate(
			numeric.RandomRange(col, col+1)*cell.X,
			numeric.RandomRange(row, row+1)*cell.Y,
		)))
	}

	return stars
}
