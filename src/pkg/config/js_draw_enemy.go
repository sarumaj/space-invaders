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
	ShapeArrow    = "arrow"    // Rank and file: a compact fighter.
	ShapeChevron  = "chevron"  // The goodie, drawn facing away from the player.
	ShapeCrown    = "crown"    // A capital ship under a crown of spires.
	ShapeDagger   = "dagger"   // A blade with wings swept back along it.
	ShapeFortress = "fortress" // A blocky hull with turrets on its back.
	ShapeHexpod   = "hexpod"   // A hexagon slung between two engine pods.
	ShapePhantom  = "phantom"  // A hollow lozenge that barely registers.
	ShapePrism    = "prism"    // A faceted crystal.
	ShapeRam      = "ram"      // A heavy wedge behind a battering prow.
	ShapeSaucer   = "saucer"   // A wide disc under a dome.
	ShapeShield   = "shield"   // A braced shield plate.
	ShapeSpike    = "spike"    // A blunt head with three fangs.
	ShapeTrident  = "trident"  // Three long prongs on a broad bar.
)

// hullOutline is the colour every hull is outlined with, which is what keeps a
// dark hull legible against the starfield.
const hullOutline = "rgba(0, 0, 0, 0.85)"

// parseRGBA splits a colour of the form "rgba(r, g, b, a)" into its components.
// Enemy colours reach the painters in exactly that form, from Color.FormatRGBA.
func parseRGBA(color string) (r, g, b int, a float64, ok bool) {
	if _, err := fmt.Sscanf(color, "rgba(%d, %d, %d, %f)", &r, &g, &b, &a); err != nil {
		return 255, 255, 255, 1, false
	}

	return r, g, b, a, true
}

// mixRGB moves a colour towards white for a positive amount and towards black
// for a negative one, keeping its alpha. It is what gives the flat fills a lit
// side and a shaded side.
func mixRGB(color string, amount float64) string {
	r, g, b, a, ok := parseRGBA(color)
	if !ok {
		return color
	}

	towards := 0.0
	if amount > 0 {
		towards = 255
	}

	blend := func(c int) int {
		return int(float64(c) + (towards-float64(c))*math.Abs(amount))
	}

	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", blend(r), blend(g), blend(b), a)
}

// tracePolygon lays down the path of a polygon without painting it.
func tracePolygon(points [][2]float64) {
	if len(points) < 3 {
		return
	}

	drawTarget.Call("beginPath")
	drawTarget.Call("moveTo", points[0][0], points[0][1])
	for _, point := range points[1:] {
		drawTarget.Call("lineTo", point[0], point[1])
	}
	drawTarget.Call("closePath")
}

// fillPolygon fills the polygon through the given points and outlines it.
func fillPolygon(points [][2]float64, color string) {
	if len(points) < 3 {
		return
	}

	tracePolygon(points)
	drawTarget.Set("fillStyle", color)
	drawTarget.Call("fill")

	drawTarget.Set("strokeStyle", hullOutline)
	drawTarget.Call("stroke")
}

// fillPlate fills the polygon with a vertical gradient running from a lit top to
// a shaded bottom, and outlines it. Flat fills read as paper cut-outs; the
// gradient is what makes a hull look like a solid object at this size.
func fillPlate(points [][2]float64, color string, top, height float64) {
	if len(points) < 3 || height <= 0 {
		fillPolygon(points, color)
		return
	}

	gradient := drawTarget.Call("createLinearGradient", 0, top, 0, top+height)
	gradient.Call("addColorStop", 0, mixRGB(color, 0.45))
	gradient.Call("addColorStop", 0.45, color)
	gradient.Call("addColorStop", 1, mixRGB(color, -0.45))

	tracePolygon(points)
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	drawTarget.Set("strokeStyle", hullOutline)
	drawTarget.Call("stroke")
}

// fillEllipse fills an axis-aligned ellipse and outlines it.
func fillEllipse(cx, cy, rx, ry float64, color string) {
	drawTarget.Call("beginPath")
	drawTarget.Call("ellipse", cx, cy, rx, ry, 0, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")

	drawTarget.Set("fillStyle", color)
	drawTarget.Call("fill")

	drawTarget.Set("strokeStyle", hullOutline)
	drawTarget.Call("stroke")
}

// drawEnemyHull paints the given shape into the box at x, y.
// The hulls all point down the screen, towards the player, except the chevron,
// which points away because it marks the one enemy that helps.
func drawEnemyHull(shape string, x, y, w, h float64, color string) {
	// px and py map the unit square onto the box, so the geometry below reads as
	// fractions of the hull rather than as pixel arithmetic.
	px := func(u float64) float64 { return x + w*u }
	py := func(v float64) float64 { return y + h*v }
	poly := func(pts ...[2]float64) [][2]float64 {
		out := make([][2]float64, 0, len(pts))
		for _, p := range pts {
			out = append(out, [2]float64{px(p[0]), py(p[1])})
		}
		return out
	}

	switch shape {
	case ShapeChevron:
		// Broad arrowhead pointing up and away from the player.
		fillPlate(poly(
			[2]float64{0.5, 0.02}, [2]float64{0.98, 0.60}, [2]float64{0.72, 0.60},
			[2]float64{0.72, 0.98}, [2]float64{0.28, 0.98}, [2]float64{0.28, 0.60},
			[2]float64{0.02, 0.60},
		), color, y, h)
		fillPolygon(poly(
			[2]float64{0.5, 0.18}, [2]float64{0.74, 0.52}, [2]float64{0.26, 0.52},
		), mixRGB(color, 0.55))

	case ShapePrism:
		// Faceted crystal: a six-sided body split down the middle so the two
		// halves catch the light differently.
		body := poly(
			[2]float64{0.5, 0.02}, [2]float64{0.90, 0.30}, [2]float64{0.90, 0.66},
			[2]float64{0.5, 0.98}, [2]float64{0.10, 0.66}, [2]float64{0.10, 0.30},
		)
		fillPlate(body, color, y, h)
		fillPolygon(poly(
			[2]float64{0.5, 0.02}, [2]float64{0.90, 0.30}, [2]float64{0.90, 0.66},
			[2]float64{0.5, 0.98},
		), mixRGB(color, -0.35))
		fillPolygon(poly(
			[2]float64{0.5, 0.14}, [2]float64{0.66, 0.34}, [2]float64{0.5, 0.72},
			[2]float64{0.34, 0.34},
		), mixRGB(color, 0.7))

	case ShapePhantom:
		// A hollow lozenge inside a faint aura: the hull is meant to be hard to
		// pick out, but it has to be large enough to be findable at all.
		fillPolygon(poly(
			[2]float64{0.5, 0.00}, [2]float64{1.00, 0.50}, [2]float64{0.5, 1.00},
			[2]float64{0.00, 0.50},
		), mixRGB(color, -0.55))
		fillPlate(poly(
			[2]float64{0.5, 0.10}, [2]float64{0.86, 0.50}, [2]float64{0.5, 0.90},
			[2]float64{0.14, 0.50},
		), color, y, h)
		fillPolygon(poly(
			[2]float64{0.5, 0.32}, [2]float64{0.66, 0.50}, [2]float64{0.5, 0.68},
			[2]float64{0.34, 0.50},
		), "rgba(0, 0, 0, 0.65)")

	case ShapeSpike:
		// A blunt head with three fangs hanging off it. Short and fat, so that it
		// does not read as the trident's long prongs.
		fillPlate(poly(
			[2]float64{0.04, 0.06}, [2]float64{0.96, 0.06}, [2]float64{0.88, 0.52},
			[2]float64{0.12, 0.52},
		), color, y, h*0.6)
		for _, u := range [...]float64{0.22, 0.5, 0.78} {
			fillPolygon(poly(
				[2]float64{u - 0.14, 0.50}, [2]float64{u + 0.14, 0.50}, [2]float64{u, 0.96},
			), mixRGB(color, -0.25))
		}
		for _, u := range [...]float64{0.32, 0.68} {
			fillEllipse(px(u), py(0.26), w*0.07, h*0.07, "rgba(0, 0, 0, 0.7)")
		}

	case ShapeHexpod:
		// Hexagonal core slung between two engine pods that clear the hull, so
		// the pods stay part of the silhouette rather than merging into it.
		for _, u := range [...]float64{0.11, 0.89} {
			fillEllipse(px(u), py(0.46), w*0.11, h*0.34, mixRGB(color, -0.3))
			fillEllipse(px(u), py(0.74), w*0.06, h*0.09, mixRGB(color, 0.5))
		}
		fillPlate(poly(
			[2]float64{0.5, 0.02}, [2]float64{0.78, 0.26}, [2]float64{0.78, 0.72},
			[2]float64{0.5, 0.98}, [2]float64{0.22, 0.72}, [2]float64{0.22, 0.26},
		), color, y, h)
		fillEllipse(px(0.5), py(0.42), w*0.13, h*0.15, mixRGB(color, 0.6))

	case ShapeRam:
		// Heavy wedge behind a battering prow.
		fillPlate(poly(
			[2]float64{0.06, 0.04}, [2]float64{0.94, 0.04}, [2]float64{1.00, 0.48},
			[2]float64{0.5, 0.80}, [2]float64{0.00, 0.48},
		), color, y, h)
		fillPolygon(poly(
			[2]float64{0.30, 0.70}, [2]float64{0.70, 0.70}, [2]float64{0.5, 1.00},
		), mixRGB(color, 0.5))
		fillPolygon(poly(
			[2]float64{0.16, 0.16}, [2]float64{0.84, 0.16}, [2]float64{0.78, 0.34},
			[2]float64{0.22, 0.34},
		), mixRGB(color, -0.4))

	case ShapeDagger:
		// A blade with its wings swept back along the hull. The wings used to
		// splay sideways, which made the whole thing read as an arrow pointing
		// left rather than as a ship pointing down.
		fillPolygon(poly(
			[2]float64{0.40, 0.24}, [2]float64{0.40, 0.70}, [2]float64{0.10, 0.34},
			[2]float64{0.14, 0.06},
		), mixRGB(color, -0.3))
		fillPolygon(poly(
			[2]float64{0.60, 0.24}, [2]float64{0.60, 0.70}, [2]float64{0.90, 0.34},
			[2]float64{0.86, 0.06},
		), mixRGB(color, -0.3))
		fillPlate(poly(
			[2]float64{0.5, 0.00}, [2]float64{0.62, 0.16}, [2]float64{0.62, 0.74},
			[2]float64{0.5, 1.00}, [2]float64{0.38, 0.74}, [2]float64{0.38, 0.16},
		), color, y, h)
		fillEllipse(px(0.5), py(0.34), w*0.07, h*0.12, mixRGB(color, 0.65))

	case ShapeFortress:
		// Blocky armoured hull with turrets standing clear of its back.
		for _, u := range [...]float64{0.22, 0.5, 0.78} {
			fillPolygon(poly(
				[2]float64{u - 0.08, 0.00}, [2]float64{u + 0.08, 0.00},
				[2]float64{u + 0.08, 0.22}, [2]float64{u - 0.08, 0.22},
			), mixRGB(color, -0.35))
		}
		fillPlate(poly(
			[2]float64{0.04, 0.20}, [2]float64{0.96, 0.20}, [2]float64{0.96, 0.68},
			[2]float64{0.72, 0.96}, [2]float64{0.28, 0.96}, [2]float64{0.04, 0.68},
		), color, py(0.2), h*0.76)
		fillPolygon(poly(
			[2]float64{0.14, 0.46}, [2]float64{0.86, 0.46}, [2]float64{0.86, 0.58},
			[2]float64{0.14, 0.58},
		), mixRGB(color, -0.45))

	case ShapeShield:
		// A braced shield plate: the Bulwark used to borrow the fortress hull, so
		// two different types looked identical apart from their colour.
		fillPlate(poly(
			[2]float64{0.08, 0.06}, [2]float64{0.92, 0.06}, [2]float64{0.92, 0.52},
			[2]float64{0.5, 0.98}, [2]float64{0.08, 0.52},
		), color, y, h)
		fillPolygon(poly(
			[2]float64{0.44, 0.14}, [2]float64{0.56, 0.14}, [2]float64{0.56, 0.80},
			[2]float64{0.44, 0.80},
		), mixRGB(color, 0.55))
		fillPolygon(poly(
			[2]float64{0.18, 0.36}, [2]float64{0.82, 0.36}, [2]float64{0.82, 0.48},
			[2]float64{0.18, 0.48},
		), mixRGB(color, 0.55))

	case ShapeTrident:
		// Three long prongs hanging off a broad bar. The prongs are deliberately
		// thin and long where the spike's fangs are short and fat.
		fillPlate(poly(
			[2]float64{0.00, 0.04}, [2]float64{1.00, 0.04}, [2]float64{0.90, 0.32},
			[2]float64{0.10, 0.32},
		), color, y, h*0.4)
		for _, u := range [...]float64{0.18, 0.5, 0.82} {
			fillPolygon(poly(
				[2]float64{u - 0.07, 0.30}, [2]float64{u + 0.07, 0.30},
				[2]float64{u + 0.04, 0.86}, [2]float64{u, 1.00}, [2]float64{u - 0.04, 0.86},
			), mixRGB(color, -0.2))
		}

	case ShapeSaucer:
		// Wide disc under a raised dome.
		fillEllipse(px(0.5), py(0.58), w*0.5, h*0.26, mixRGB(color, -0.3))
		fillEllipse(px(0.5), py(0.50), w*0.44, h*0.18, color)
		fillEllipse(px(0.5), py(0.34), w*0.24, h*0.20, mixRGB(color, 0.35))
		fillEllipse(px(0.5), py(0.30), w*0.10, h*0.08, mixRGB(color, 0.8))

	case ShapeCrown:
		// Capital ship: a wide hull under a crown of spires, with the gaps
		// between the spires kept wide enough to survive at this size.
		for _, u := range [...]float64{0.24, 0.5, 0.76} {
			fillPolygon(poly(
				[2]float64{u - 0.09, 0.30}, [2]float64{u, 0.00}, [2]float64{u + 0.09, 0.30},
			), mixRGB(color, -0.3))
		}
		fillPlate(poly(
			[2]float64{0.02, 0.30}, [2]float64{0.98, 0.30}, [2]float64{0.84, 0.72},
			[2]float64{0.16, 0.72},
		), color, py(0.3), h*0.42)
		fillPolygon(poly(
			[2]float64{0.34, 0.72}, [2]float64{0.66, 0.72}, [2]float64{0.5, 1.00},
		), mixRGB(color, 0.4))
		fillEllipse(px(0.5), py(0.48), w*0.11, h*0.10, mixRGB(color, 0.7))

	default: // ShapeArrow
		// A compact fighter: a fuselage with wings swept back from it. This is
		// the hull the bulk of the roster wears, and as a bare dart it read as a
		// typographic arrow rather than as a ship.
		fillPolygon(poly(
			[2]float64{0.36, 0.28}, [2]float64{0.36, 0.62}, [2]float64{0.02, 0.44},
			[2]float64{0.08, 0.14},
		), mixRGB(color, -0.35))
		fillPolygon(poly(
			[2]float64{0.64, 0.28}, [2]float64{0.64, 0.62}, [2]float64{0.98, 0.44},
			[2]float64{0.92, 0.14},
		), mixRGB(color, -0.35))
		fillPlate(poly(
			[2]float64{0.5, 0.02}, [2]float64{0.66, 0.24}, [2]float64{0.64, 0.70},
			[2]float64{0.5, 1.00}, [2]float64{0.36, 0.70}, [2]float64{0.34, 0.24},
		), color, y, h)
		fillEllipse(px(0.5), py(0.36), w*0.10, h*0.13, mixRGB(color, 0.7))
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
