//go:build js && wasm

package config

import (
	"fmt"
	"math"
)

// Enemy hull shapes.
// Every enemy type used to be drawn with the player's own silhouette in a
// different colour, so a Freezer and a Dreadnought were told apart only by a hue,
// and several of those hues were near black against the starfield. Each type now
// has an outline the player can recognise before reading its colour.
const (
	ShapeArrow    = "arrow"    // Rank and file: a small dart.
	ShapeChevron  = "chevron"  // The goodie, drawn facing away from the player.
	ShapeCrown    = "crown"    // A crowned capital ship.
	ShapeDagger   = "dagger"   // A long blade with swept wings.
	ShapeFortress = "fortress" // A blocky hull with turret notches.
	ShapeHexpod   = "hexpod"   // A hexagon flanked by engine pods.
	ShapePhantom  = "phantom"  // A hollow lozenge that barely registers.
	ShapePrism    = "prism"    // A faceted crystal.
	ShapeRam      = "ram"      // A heavy wedge behind a battering prow.
	ShapeSaucer   = "saucer"   // A wide disc under a dome.
	ShapeSpike    = "spike"    // Three forward spikes.
	ShapeTrident  = "trident"  // Three prongs on a broad base.
)

// fillPolygon fills the polygon through the given points and outlines it.
// The outline is what keeps a dark hull legible against the starfield.
func fillPolygon(points [][2]float64, color string) {
	if len(points) < 3 {
		return
	}

	drawTarget.Call("beginPath")
	drawTarget.Call("moveTo", points[0][0], points[0][1])
	for _, point := range points[1:] {
		drawTarget.Call("lineTo", point[0], point[1])
	}
	drawTarget.Call("closePath")

	drawTarget.Set("fillStyle", color)
	drawTarget.Call("fill")

	drawTarget.Set("strokeStyle", "rgba(0, 0, 0, 0.85)")
	drawTarget.Call("stroke")
}

// fillEllipse fills an axis-aligned ellipse and outlines it.
func fillEllipse(cx, cy, rx, ry float64, color string) {
	drawTarget.Call("beginPath")
	drawTarget.Call("ellipse", cx, cy, rx, ry, 0, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")

	drawTarget.Set("fillStyle", color)
	drawTarget.Call("fill")

	drawTarget.Set("strokeStyle", "rgba(0, 0, 0, 0.85)")
	drawTarget.Call("stroke")
}

// drawEnemyHull paints the outline of the given shape into the box at x, y.
// The shapes all point down the screen, towards the player, except the chevron,
// which points away because it marks the one enemy that helps.
func drawEnemyHull(shape string, x, y, w, h float64, color string) {
	switch shape {
	case ShapeChevron:
		// Broad arrowhead pointing up and away from the player.
		fillPolygon([][2]float64{
			{x + w*0.5, y},
			{x + w, y + h*0.62},
			{x + w*0.72, y + h*0.62},
			{x + w*0.72, y + h},
			{x + w*0.28, y + h},
			{x + w*0.28, y + h*0.62},
			{x, y + h*0.62},
		}, color)

	case ShapePrism:
		// Faceted crystal: two mirrored halves with a bright core sliver.
		fillPolygon([][2]float64{
			{x + w*0.5, y},
			{x + w*0.9, y + h*0.35},
			{x + w*0.5, y + h},
			{x + w*0.1, y + h*0.35},
		}, color)
		fillPolygon([][2]float64{
			{x + w*0.5, y + h*0.1},
			{x + w*0.68, y + h*0.38},
			{x + w*0.5, y + h*0.82},
			{x + w*0.32, y + h*0.38},
		}, "rgba(255, 255, 255, 0.35)")

	case ShapePhantom:
		// A hollow lozenge: the outer body is faint and only the rim is solid,
		// so the hull reads as a smudge until it is close.
		fillPolygon([][2]float64{
			{x + w*0.5, y + h*0.05},
			{x + w*0.85, y + h*0.5},
			{x + w*0.5, y + h*0.95},
			{x + w*0.15, y + h*0.5},
		}, color)
		fillPolygon([][2]float64{
			{x + w*0.5, y + h*0.3},
			{x + w*0.66, y + h*0.5},
			{x + w*0.5, y + h*0.7},
			{x + w*0.34, y + h*0.5},
		}, "rgba(0, 0, 0, 0.55)")

	case ShapeSpike:
		// Three forward spikes on a narrow spine.
		for _, offset := range [...]float64{0.16, 0.5, 0.84} {
			fillPolygon([][2]float64{
				{x + w*(offset-0.13), y + h*0.35},
				{x + w*offset, y + h},
				{x + w*(offset+0.13), y + h*0.35},
			}, color)
		}
		fillPolygon([][2]float64{
			{x, y + h*0.12},
			{x + w, y + h*0.12},
			{x + w*0.8, y + h*0.45},
			{x + w*0.2, y + h*0.45},
		}, color)

	case ShapeHexpod:
		// Hexagonal core flanked by two engine pods.
		fillPolygon([][2]float64{
			{x + w*0.5, y},
			{x + w*0.82, y + h*0.28},
			{x + w*0.82, y + h*0.72},
			{x + w*0.5, y + h},
			{x + w*0.18, y + h*0.72},
			{x + w*0.18, y + h*0.28},
		}, color)
		fillEllipse(x+w*0.12, y+h*0.5, w*0.12, h*0.26, color)
		fillEllipse(x+w*0.88, y+h*0.5, w*0.12, h*0.26, color)

	case ShapeRam:
		// Heavy wedge behind a battering prow.
		fillPolygon([][2]float64{
			{x + w*0.12, y},
			{x + w*0.88, y},
			{x + w, y + h*0.55},
			{x + w*0.5, y + h*0.78},
			{x, y + h*0.55},
		}, color)
		fillPolygon([][2]float64{
			{x + w*0.3, y + h*0.7},
			{x + w*0.7, y + h*0.7},
			{x + w*0.5, y + h},
		}, "rgba(255, 255, 255, 0.3)")

	case ShapeDagger:
		// Long blade with wings swept back along the hull.
		fillPolygon([][2]float64{
			{x + w*0.42, y},
			{x + w*0.58, y},
			{x + w*0.58, y + h*0.7},
			{x + w*0.5, y + h},
			{x + w*0.42, y + h*0.7},
		}, color)
		fillPolygon([][2]float64{
			{x + w*0.42, y + h*0.15},
			{x + w*0.42, y + h*0.6},
			{x, y + h*0.3},
		}, color)
		fillPolygon([][2]float64{
			{x + w*0.58, y + h*0.15},
			{x + w*0.58, y + h*0.6},
			{x + w, y + h*0.3},
		}, color)

	case ShapeFortress:
		// Blocky armoured hull with turret notches along the leading edge.
		fillPolygon([][2]float64{
			{x + w*0.08, y + h*0.1},
			{x + w*0.92, y + h*0.1},
			{x + w*0.92, y + h*0.72},
			{x + w*0.66, y + h*0.92},
			{x + w*0.34, y + h*0.92},
			{x + w*0.08, y + h*0.72},
		}, color)
		for _, offset := range [...]float64{0.22, 0.5, 0.78} {
			fillPolygon([][2]float64{
				{x + w*(offset-0.09), y},
				{x + w*(offset+0.09), y},
				{x + w*(offset+0.09), y + h*0.1},
				{x + w*(offset-0.09), y + h*0.1},
			}, color)
		}

	case ShapeTrident:
		// Three prongs hanging off a broad base.
		fillPolygon([][2]float64{
			{x, y + h*0.05},
			{x + w, y + h*0.05},
			{x + w*0.85, y + h*0.42},
			{x + w*0.15, y + h*0.42},
		}, color)
		for _, offset := range [...]float64{0.2, 0.5, 0.8} {
			fillPolygon([][2]float64{
				{x + w*(offset-0.1), y + h*0.42},
				{x + w*(offset+0.1), y + h*0.42},
				{x + w*offset, y + h},
			}, color)
		}

	case ShapeSaucer:
		// Wide disc under a raised dome.
		fillEllipse(x+w*0.5, y+h*0.55, w*0.5, h*0.28, color)
		fillEllipse(x+w*0.5, y+h*0.38, w*0.26, h*0.22, color)
		fillEllipse(x+w*0.5, y+h*0.34, w*0.12, h*0.1, "rgba(255, 255, 255, 0.4)")

	case ShapeCrown:
		// Capital ship: a wide hull topped with a crown of spires.
		fillPolygon([][2]float64{
			{x, y + h*0.35},
			{x + w*0.2, y + h*0.2},
			{x + w*0.8, y + h*0.2},
			{x + w, y + h*0.35},
			{x + w*0.72, y + h*0.85},
			{x + w*0.28, y + h*0.85},
		}, color)
		for _, offset := range [...]float64{0.28, 0.5, 0.72} {
			fillPolygon([][2]float64{
				{x + w*(offset-0.08), y + h*0.2},
				{x + w*offset, y},
				{x + w*(offset+0.08), y + h*0.2},
			}, color)
		}
		fillPolygon([][2]float64{
			{x + w*0.38, y + h*0.85},
			{x + w*0.62, y + h*0.85},
			{x + w*0.5, y + h},
		}, color)

	default: // ShapeArrow
		// A small dart: the shape the whole roster used to share, kept for the
		// rank and file it actually suits.
		fillPolygon([][2]float64{
			{x + w*0.5, y + h},
			{x + w*0.2, y + h*0.25},
			{x + w*0.5, y + h*0.4},
			{x + w*0.8, y + h*0.25},
		}, color)
		fillPolygon([][2]float64{
			{x + w*0.4, y},
			{x + w*0.6, y},
			{x + w*0.6, y + h*0.5},
			{x + w*0.4, y + h*0.5},
		}, color)
	}
}

// DrawEnemy draws an enemy of the given shape, with its label and status arcs.
// The flash value, from 0 to 1, whitens the hull right after a hit: damage used
// to be reported only as a number in the message log and a sound.
func DrawEnemy(coords [2]float64, size [2]float64, shape, color, label string, statusValues []float64, statusColors []string, flash float64) {
	x, y := coords[0], coords[1]
	width, height := size[0], size[1]

	defaultLineWidth := drawTarget.Get("lineWidth")
	defer drawTarget.Set("lineWidth", defaultLineWidth)
	drawTarget.Set("lineWidth", 1)

	drawEnemyHull(shape, x, y, width, height, color)

	if flash > 0 {
		drawEnemyHull(shape, x, y, width, height, fmt.Sprintf("rgba(255, 255, 255, %.2f)", flash))
	}

	// The chevron is the only hull that faces away from the player, so it is also
	// the only one whose label and status arcs belong on the far side.
	faceUp := shape == ShapeChevron
	drawObjectLabel(x, y, width, height, faceUp, color, label)
	drawStatusArcs(x, y, width, height, faceUp, statusValues, statusColors)
}
