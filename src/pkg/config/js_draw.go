//go:build js && wasm

package config

import (
	"fmt"
	"math"
	"syscall/js"
	"time"

	"math/rand"
)

// drawTarget is the canvas context the planet painters draw into.
// It is the visible canvas except while RenderSprite is capturing a planet into
// an offscreen canvas, which is safe because Go on WebAssembly runs the painters
// on a single thread.
var drawTarget = canvasObjectContext

// Sprite is an opaque handle to a pre-rendered image.
type Sprite struct{ canvas js.Value }

// Valid reports whether the sprite has been rendered and can be drawn.
func (sprite Sprite) Valid() bool { return sprite.canvas.Truthy() }

// RenderSprite paints into a square offscreen canvas of the given size and
// returns it, handing draw the centre of that canvas.
//
// Each planet is built from a dozen gradients and clip paths, none of which
// change between frames, so it is worth painting once and blitting the result
// afterwards. It also makes rotating a planet almost free.
func RenderSprite(size float64, draw func(center [2]float64)) Sprite {
	canvas := document.Call("createElement", "canvas")
	canvas.Set("width", size)
	canvas.Set("height", size)

	previous := drawTarget
	drawTarget = canvas.Call("getContext", "2d")
	defer func() { drawTarget = previous }()

	draw([2]float64{size / 2, size / 2})

	return Sprite{canvas: canvas}
}

// DrawSprite blits a sprite centred on the given coordinates, turned by the given
// angle in radians.
func DrawSprite(sprite Sprite, coords [2]float64, angle float64) {
	if !sprite.Valid() {
		return
	}

	canvasObjectContext.Call("save")
	canvasObjectContext.Call("translate", coords[0], coords[1])
	canvasObjectContext.Call("rotate", angle)
	canvasObjectContext.Call("drawImage", sprite.canvas,
		-sprite.canvas.Get("width").Float()/2, -sprite.canvas.Get("height").Float()/2)
	canvasObjectContext.Call("restore")
}

// Struct to represent color stops for gradients.
type colorStop struct {
	Position float64
	Color    string
}

// Helper function to draw a circle with a radial gradient.
func drawCircleWithGradient(cx, cy, radius float64, colorStops []colorStop) {
	gradient := drawTarget.Call("createRadialGradient", cx, cy, radius*0.3, cx, cy, radius)
	for _, stop := range colorStops {
		gradient.Call("addColorStop", stop.Position, stop.Color)
	}
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")
}

// Helper function to draw an arc.
func drawArc(cx, cy, radius float64, color string) {
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Set("fillStyle", color)
	drawTarget.Call("fill")
}

// Helper function to create a radial gradient with color stops.
func createRadialGradient(cx, cy, innerRadius, outerRadius float64, colorStops []colorStop) interface{} {
	gradient := drawTarget.Call("createRadialGradient", cx, cy, innerRadius, cx, cy, outerRadius)
	for _, stop := range colorStops {
		gradient.Call("addColorStop", stop.Position, stop.Color)
	}
	return gradient
}

// DrawAnomalyBlackHole draws a black hole on the document.
func DrawAnomalyBlackHole(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]
	scale := 1.0 + rand.Float64()/10

	// Clear a larger area to enhance the effect of the black hole
	clearRadius := radius * 1.3
	drawTarget.Call("clearRect", cx-clearRadius, cy-clearRadius, clearRadius*2, clearRadius*2)

	// Draw the dark core of the black hole
	drawArc(cx, cy, scale*0.6*radius, "black")

	// Draw a subtle glow around the black hole to simulate light bending
	glowGradient := createRadialGradient(cx, cy, scale*0.6*radius, scale*radius, []colorStop{
		{0, "rgba(0, 0, 0, 0.0)"},
		{1, "rgba(0, 0, 0, 0.3)"},
	})
	drawTarget.Set("fillStyle", glowGradient)
	drawArc(cx, cy, scale*radius, "")

	// Draw the accretion disk around the black hole
	gradient := createRadialGradient(cx, cy, scale*0.6*radius, scale*radius, []colorStop{
		{0, "rgba(0, 0, 0, 0.0)"},
		{0.15, "rgba(128, 0, 128, 0.2)"},
		{0.35, "rgba(78, 0, 78, 0.6)"},
		{0.6, "rgba(128, 0, 78, 0.8)"},
		{0.8, "rgba(128, 0, 128, 0.6)"},
		{1, "rgba(0, 0, 0, 0.0)"},
	})
	drawTarget.Set("fillStyle", gradient)
	drawArc(cx, cy, scale*radius, "")
}

// DrawAnomalySupernova draws a supernova on the document.
func DrawAnomalySupernova(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]
	scale := 1.0 + rand.Float64()/10

	// Draw the epicenter using a radial gradient
	epicenterGradient := createRadialGradient(cx, cy, radius*0.1, radius, []colorStop{
		{0, "rgba(255, 255, 255, 1)"},
		{0.3, "rgba(255, 215, 0, 0.9)"},
		{0.6, "rgba(255, 165, 0, 0.4)"},
		{0.8, "rgba(255, 69, 0, 0.2)"},
		{1, "rgba(128, 0, 128, 0.0)"},
	})
	drawTarget.Set("fillStyle", epicenterGradient)
	drawArc(cx, cy, radius, "")

	// Draw the first shockwave as a ring around the epicenter
	firstShockwaveGradient := createRadialGradient(cx, cy, radius*1.1, radius*1.5, []colorStop{
		{0, "rgba(255, 69, 0, 0.0)"},
		{0.5, "rgba(255, 140, 0, 0.4)"},
		{1, "rgba(255, 255, 0, 0.6)"},
	})
	drawTarget.Set("fillStyle", firstShockwaveGradient)
	drawArc(cx, cy, scale*radius*1.5, "")

	// Draw the second shockwave as a larger ring further from the epicenter
	secondShockwaveGradient := createRadialGradient(cx, cy, radius*1.7, radius*2.2, []colorStop{
		{0, "rgba(255, 69, 0, 0.0)"},
		{0.5, "rgba(255, 140, 0, 0.3)"},
		{1, "rgba(255, 255, 255, 0.4)"},
	})
	drawTarget.Set("fillStyle", secondShockwaveGradient)
	drawArc(cx, cy, scale*radius*2.2, "")
}

// DrawBackground is a function that draws the background of the document.
// The background is drawn with the specified speed.
func DrawBackground(speed float64) {
	if !*Config.Control.BackgroundAnimationEnabled {
		canvasObjectContext.Call("drawImage", invisibleCanvas, 0, 0)
		return
	}

	canvasDimensions := CanvasBoundingBox()

	// Apply the speed
	invisibleCanvasScrollY += speed
	if invisibleCanvasScrollY/canvasDimensions.OriginalHeight > 1 {
		invisibleCanvasScrollY -= canvasDimensions.OriginalHeight
	}

	canvasObjectContext.Call("drawImage", invisibleCanvas, 0, invisibleCanvasScrollY)
	canvasObjectContext.Call("drawImage", invisibleCanvas, 0, invisibleCanvasScrollY-canvasDimensions.OriginalHeight)
}

// DrawLine is a function that draws a line on the document.
func DrawLine(start, end [2]float64, color string, thickness float64) {
	defaultLineWidth := canvasObjectContext.Get("lineWidth")
	defer canvasObjectContext.Set("lineWidth", defaultLineWidth)

	canvasObjectContext.Set("strokeStyle", color)
	canvasObjectContext.Set("lineWidth", thickness)
	canvasObjectContext.Call("beginPath")
	canvasObjectContext.Call("moveTo", start[0], start[1])
	canvasObjectContext.Call("lineTo", end[0], end[1])
	canvasObjectContext.Call("stroke")
}

// DrawPlanetEarth is a function that draws Earth on the document.
func DrawPlanetEarth(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	// Begin drawing the Earth
	drawArc(cx, cy, radius, "")

	// Use a blue gradient to represent the oceans
	gradient := createRadialGradient(cx, cy, radius*0.2, radius, []colorStop{
		{0, "#00BFFF"},
		{1, "#1E90FF"},
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Add an atmospheric glow around the Earth
	atmosphereGradient := createRadialGradient(cx, cy, radius, radius*1.2, []colorStop{
		{0, "rgba(30, 144, 255, 0.1)"},
		{1, "rgba(30, 144, 255, 0.0)"},
	})
	drawTarget.Set("fillStyle", atmosphereGradient)
	drawArc(cx, cy, radius*1.2, "")

	// Clip to the planet's circle to restrict drawing within the Earth
	drawTarget.Call("save")
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip") // Apply clipping here before drawing the north pole and other elements

	// Draw the north pole
	poleRadiusInner := radius * 0.15
	poleRadiusOuter := radius * 0.25
	rotationAngle := math.Pi / 12

	drawTarget.Call("beginPath")
	drawTarget.Call("ellipse", cx, cy-radius, poleRadiusOuter, poleRadiusInner, rotationAngle, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Set("fillStyle", "#FFFFFF") // White for the pole
	drawTarget.Call("fill")

	// Add more complex land masses with gradients for a realistic look
	landColors := []string{"#228B22", "#8B4513"}
	landPatches := [][5]float64{
		{cx - radius*0.2, cy - radius*0.3, radius * 0.4, radius * 0.35, math.Pi / 45},
		{cx + radius*0.1, cy + radius*0.2, radius * 0.35, radius * 0.3, math.Pi / 30},
		{cx + radius*0.15, cy - radius*0.1, radius * 0.25, radius * 0.4, math.Pi / 60},
		{cx + radius*0.85, cy + radius*0.2, radius * 0.3, radius * 0.25, math.Pi / 40},
	}

	for i, patch := range landPatches {
		drawTarget.Call("beginPath")
		drawTarget.Call("ellipse", patch[0], patch[1], patch[2], patch[3], patch[4], 0, 2*math.Pi, false)
		drawTarget.Call("closePath")

		landGradient := createRadialGradient(patch[0], patch[1], patch[2]*0.5, patch[2], []colorStop{
			{0, landColors[i%len(landColors)]},
			{1, "#556B2F"},
		})
		drawTarget.Set("fillStyle", landGradient)
		drawTarget.Call("fill")
	}

	// Add more dynamic clouds with some variation
	clouds := [][4]float64{
		{cx - radius*0.4, cy - radius*0.1, radius * 0.6, radius * 0.2},
		{cx + radius*0.3, cy + radius*0.2, radius * 0.5, radius * 0.25},
		{cx - radius*0.2, cy + radius*0.1, radius * 0.4, radius * 0.15},
	}

	for _, cloud := range clouds {
		drawTarget.Call("beginPath")
		drawTarget.Call("ellipse", cloud[0], cloud[1], cloud[2], cloud[3], 0, 0, 2*math.Pi, false)
		drawTarget.Call("closePath")

		cloudGradient := createRadialGradient(cloud[0], cloud[1], cloud[2]*0.5, cloud[2], []colorStop{
			{0, "rgba(255, 255, 255, 0.8)"},
			{1, "rgba(255, 255, 255, 0.4)"},
		})
		drawTarget.Set("fillStyle", cloudGradient)
		drawTarget.Call("fill")
	}

	drawTarget.Call("restore") // Restore the drawing state, removing the clip

	// Draw the Moon orbiting Earth
	moonRadius := radius * 0.27
	moonDistance := radius * 60.3 / 30

	// Calculate the moon's current position based on phase
	const siderealMonth = 27.321661
	referenceTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	elapsedDays := time.Now().UTC().Sub(referenceTime).Hours() / 24
	phase := (elapsedDays / siderealMonth) * 2 * math.Pi
	phase = math.Mod(phase, 2*math.Pi)

	moonX := cx + moonDistance*math.Cos(phase)
	moonY := cy + moonDistance*math.Sin(phase)

	drawArc(moonX, moonY, moonRadius, "#F0F0F0")

	// Add a crater to the Moon
	craterX := moonX + moonRadius*0.2 // Position the crater slightly offset from the Moon's center
	craterY := moonY + moonRadius*0.1
	craterRadius := moonRadius * 0.3 // Crater is 30% the size of the Moon

	drawTarget.Call("save") // Save the drawing state to clip the Moon
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", craterX, craterY, craterRadius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip") // Clip to the Moon's circle

	drawArc(craterX, craterY, craterRadius, "#A9A9A9")

	drawTarget.Call("restore") // Restore the drawing state to remove the clipping
}

// DrawPlanetJupiter is a function that draws Jupiter on the document.
func DrawPlanetJupiter(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1] // Center position

	// Draw the main body of Jupiter (a sphere)
	drawArc(cx, cy, radius, "")

	// Create a radial gradient to simulate the planet's lighting and subtle pole banding
	gradient := createRadialGradient(cx, cy, radius*0.3, radius, []colorStop{
		{0, "#FFF4C3"},
		{0.7, "#E2B56D"},
		{1, "#B58A4C"},
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Clip the drawing area to the circle of the planet
	drawTarget.Call("save") // Save the current drawing state
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip") // Clip to the planet's circle

	// Add bands to simulate Jupiter's gas bands
	bandColors := []string{
		"rgba(243, 210, 158, 0.7)", // #F3D29E (Pale Goldenrod) with 70% opacity
		"rgba(234, 178, 119, 0.7)", // #EAB277 (Sandy Brown) with 70% opacity
		"rgba(229, 170, 102, 0.7)", // #E5AA66 (Light Salmon) with 70% opacity
		"rgba(223, 154, 85, 0.7)",  // #DF9A55 (Moccasin) with 70% opacity
		"rgba(217, 138, 68, 0.7)",  // #D98A44 (Dark Salmon) with 70% opacity
		"rgba(208, 122, 51, 0.7)",  // #D07A33 (Chocolate) with 70% opacity
		"rgba(201, 105, 34, 0.7)",  // #C96922 (Peru) with 70% opacity
		"rgba(194, 88, 17, 0.7)",   // #C25811 (Sienna) with 70% opacity
		"rgba(187, 71, 0, 0.7)",    // #BB4700 (Dark Orange) with 70% opacity

	}
	bandHeight := (radius * 2) / float64(len(bandColors))

	for i, color := range bandColors {
		y := cy - radius + float64(i)*bandHeight

		drawTarget.Call("beginPath")
		drawTarget.Call("rect", cx-radius, y, radius*2, bandHeight)
		drawTarget.Set("fillStyle", color)
		drawTarget.Call("fill")
		drawTarget.Call("closePath")
	}

	// Add the Great Red Spot (simply a circle here)
	drawArc(cx+radius*0.5, cy+radius*0.4, radius*0.3, "")

	redSpotGradient := createRadialGradient(cx+radius*0.5, cy+radius*0.4, 0, radius*0.3, []colorStop{
		{0, "#8B0000"},
		{0.75, "#CD5C5C"},
		{1, "#FF6347"},
	})
	drawTarget.Set("fillStyle", redSpotGradient)
	drawTarget.Call("fill")

	drawTarget.Call("restore") // Restore the drawing state to remove the clipping
}

// DrawPlanetMars is a function that draws Mars on the document.
func DrawPlanetMars(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	drawArc(cx, cy, radius, "")

	// Use a reddish color to represent Mars
	gradient := createRadialGradient(cx, cy, radius*0.2, radius, []colorStop{
		{0, "#FF7F50"},
		{0.5, "#FF6347"},
		{1, "#8B0000"},
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Darker patch representing a region like Syrtis Major
	drawArc(cx-radius*0.2, cy-radius*0.1, radius*0.3, "")

	patchGradient := createRadialGradient(cx-radius*0.2, cy-radius*0.1, 0, radius*0.3, []colorStop{
		{0, "#8B0000"},
		{1, "#A52A2A"},
	})
	drawTarget.Set("fillStyle", patchGradient)
	drawTarget.Call("fill")

	// Clip to the planet's circle to restrict the features within Mercury's shape
	drawTarget.Call("save")
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip") // Apply clipping here before drawing craters

	// Draw crater-like features on Mars with shading
	craters := [][3]float64{
		{cx - radius*0.3, cy - radius*0.3, radius * 0.1},
		{cx + radius*0.2, cy - radius*0.1, radius * 0.15},
		{cx, cy + radius*0.3, radius * 0.05},
	}

	for _, crater := range craters {
		drawArc(crater[0], crater[1], crater[2], "")

		craterGradient := createRadialGradient(crater[0], crater[1], 0, crater[2], []colorStop{
			{0, "#8B4513"},
			{0.8, "#8B4513"},
			{1, "#A0522D"},
		})
		drawTarget.Set("fillStyle", craterGradient)
		drawTarget.Call("fill")
	}

	drawTarget.Call("restore") // Restore the drawing state, removing the clip
}

// DrawPlanetMercury is a function that draws Mercury on the document.
func DrawPlanetMercury(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	drawArc(cx, cy, radius, "")

	// Use a simple gray gradient to represent Mercury
	gradient := createRadialGradient(cx, cy, radius*0.2, radius, []colorStop{
		{0, "#C0C0C0"},
		{0.7, "#A9A9A9"},
		{1, "#808080"},
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Clip to the planet's circle to restrict the features within Mercury's shape
	drawTarget.Call("save")
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip") // Apply clipping here before drawing craters

	// Draw crater-like features on Mercury with shading
	craters := [][3]float64{
		{cx - radius*0.3, cy - radius*0.3, radius * 0.1},
		{cx + radius*0.2, cy - radius*0.1, radius * 0.15},
		{cx, cy + radius*0.3, radius * 0.05},
	}

	for _, crater := range craters {
		drawArc(crater[0], crater[1], crater[2], "")

		craterGradient := createRadialGradient(crater[0], crater[1], 0, crater[2], []colorStop{
			{0, "#696969"},
			{0.9, "#A0A0A0"},
			{1, "rgba(160, 160, 160, 0)"},
		})
		drawTarget.Set("fillStyle", craterGradient)
		drawTarget.Call("fill")
	}

	drawTarget.Call("restore") // Restore the drawing state, removing the clip
}

// DrawPlanetNeptune is a function that draws Neptune on the document.
func DrawPlanetNeptune(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	drawArc(cx, cy, radius, "")

	// Deep blue color for Neptune
	gradient := createRadialGradient(cx, cy, radius*0.3, radius, []colorStop{
		{0, "#4682B4"},
		{0.5, "#4169E1"},
		{1, "#00008B"},
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Clip to the planet's circle to restrict the gas bands within Neptune's shape
	drawTarget.Call("save") // Save the current drawing state before clipping
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip")

	// Add gas bands
	bandColors := []string{
		"rgba(100, 149, 237, 0.7)", // #6495ED (Cornflower Blue) with 70% opacity
		"rgba(70, 130, 180, 0.7)",  // #4682B4 (Steel Blue) with 70% opacity
		"rgba(30, 144, 255, 0.7)",  // #1E90FF (Dodger Blue) with 70% opacity
		"rgba(135, 206, 250, 0.7)", // #87CEFA (Light Sky Blue) with 70% opacity
	}
	bandHeight := radius * 2 / float64(len(bandColors)) // Height of each band

	for i, color := range bandColors {
		y := cy - radius + float64(i)*bandHeight

		drawTarget.Call("beginPath")
		drawTarget.Call("rect", cx-radius, y, radius*2, bandHeight)
		drawTarget.Set("fillStyle", color)
		drawTarget.Call("fill")
		drawTarget.Call("closePath")
	}

	// Optionally, add a dark spot to represent one of Neptune's storms
	drawTarget.Call("beginPath")
	drawTarget.Call("ellipse", cx+radius*0.3, cy-radius*0.2, radius*0.2, radius*0.1, math.Pi/4, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Set("fillStyle", "rgba(0, 0, 139, 0.8)") // Dark blue spot
	drawTarget.Call("fill")

	drawTarget.Call("restore") // Restore the drawing state, removing the clip
}

// DrawPlanetPluto is a function that draws Pluto on the document.
func DrawPlanetPluto(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	// Draw the main body of Pluto with a gradient to simulate the icy surface
	drawArc(cx, cy, radius, "")

	// Create a gradient to represent Pluto's surface with icy and rocky textures
	gradient := createRadialGradient(cx, cy, radius*0.2, radius, []colorStop{
		{0, "#E8E8E8"},   // Light Gray for the center
		{0.5, "#C0C0C0"}, // Silver for mid-range
		{1, "#A9A9A9"},   // Dark Gray at the edges
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Clip to the planet's circle to restrict drawing within Pluto's shape
	drawTarget.Call("save") // Save the current drawing state before clipping
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip")

	// Define static craters with fixed positions and sizes
	craters := [][4]float64{
		{cx - radius*0.3, cy - radius*0.2, radius * 0.12, 0}, // x, y, size, rotation (not used)
		{cx + radius*0.2, cy + radius*0.1, radius * 0.15, 0},
		{cx - radius*0.15, cy + radius*0.25, radius * 0.08, 0},
		{cx + radius*0.35, cy - radius*0.3, radius * 0.1, 0},
		{cx, cy - radius*0.35, radius * 0.18, 0},
	}

	craterColors := []string{
		"#B0B0B0", // Light Gray
		"#A9A9A9", // Dark Gray
		"#8B8B8B", // Gray
	}

	// Draw the static craters
	for _, crater := range craters {
		drawArc(crater[0], crater[1], crater[2], "")
		drawTarget.Set("fillStyle", craterColors[int(crater[3])%len(craterColors)]) // Use fixed color
		drawTarget.Call("fill")
	}

	drawTarget.Call("restore") // Restore the drawing state, removing the clip
}

// DrawPlanetSaturn is a function that draws Saturn on the document.
func DrawPlanetSaturn(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	// Define ring properties
	innerRingRadius := radius * 1.2
	outerRingRadius := radius * 2.0
	ringTiltAngle := math.Pi / 6
	ringThickness := radius * 0.15

	// Save the context and rotate for the ring's tilt
	drawTarget.Call("save")
	drawTarget.Call("translate", cx, cy)
	drawTarget.Call("rotate", ringTiltAngle)
	drawTarget.Call("translate", -cx, -cy)

	// Draw the upper half of the rings
	for i := 0; i < 3; i++ {
		// Clip the lower half of the ellipse to draw only the upper half
		drawTarget.Call("save")
		drawTarget.Call("beginPath")
		drawTarget.Call("rect", cx-outerRingRadius, cy-outerRingRadius, 2*outerRingRadius, outerRingRadius)
		drawTarget.Call("closePath")
		drawTarget.Call("clip")

		// Draw the full ellipse, but only the upper half will be visible due to clipping
		drawTarget.Call("beginPath")
		drawTarget.Call("ellipse", cx, cy, outerRingRadius, innerRingRadius*0.4, 0, 0, 2*math.Pi, false)
		drawTarget.Call("closePath")
		drawTarget.Set("fillStyle", "rgba(210, 180, 140, 0.7)") // Consistent opacity for rings
		drawTarget.Call("fill")

		// Restore to remove clipping
		drawTarget.Call("restore")

		innerRingRadius += ringThickness
		outerRingRadius += ringThickness * 1.5
	}

	// Restore context before drawing the planet's body
	drawTarget.Call("restore")

	{
		// Draw Saturn's body
		drawArc(cx, cy, radius, "")

		gradient := createRadialGradient(cx, cy, radius*0.3, radius, []colorStop{
			{0, "#F5DEB3"},   // Wheat color
			{0.5, "#EDD9A3"}, // Lightened Wheat
			{1, "#DAA520"},   // Goldenrod color
		})
		drawTarget.Set("fillStyle", gradient)
		drawTarget.Call("fill")
	}

	// Save the context and rotate for the ring's tilt
	drawTarget.Call("save")
	drawTarget.Call("translate", cx, cy)
	drawTarget.Call("rotate", ringTiltAngle)
	drawTarget.Call("translate", -cx, -cy)

	// Draw the lower half of the rings
	innerRingRadius = radius * 1.2
	outerRingRadius = radius * 2.0
	for i := 0; i < 3; i++ {
		// Clip the upper half of the ellipse to draw only the lower half
		drawTarget.Call("save")
		drawTarget.Call("beginPath")
		drawTarget.Call("rect", cx-outerRingRadius, cy, 2*outerRingRadius, outerRingRadius)
		drawTarget.Call("closePath")
		drawTarget.Call("clip")

		// Draw the full ellipse, but only the lower half will be visible due to clipping
		drawTarget.Call("beginPath")
		drawTarget.Call("ellipse", cx, cy, outerRingRadius, innerRingRadius*0.4, 0, 0, 2*math.Pi, false)
		drawTarget.Call("closePath")
		drawTarget.Set("fillStyle", "rgba(210, 180, 140, 0.7)") // Same opacity as the upper half
		drawTarget.Call("fill")

		// Restore to remove clipping
		drawTarget.Call("restore")

		innerRingRadius += ringThickness
		outerRingRadius += ringThickness * 1.5
	}

	// Restore the context to remove the rotation
	drawTarget.Call("restore")
}

// DrawPlanetUranus is a function that draws Uranus on the document.
func DrawPlanetUranus(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	// Add Uranus's tilted rings
	innerRingRadius := radius * 1.4
	outerRingRadius := radius * 1.8
	ringTiltAngle := math.Pi / 6

	// Save the context and rotate for the ring's tilt
	drawTarget.Call("save")
	drawTarget.Call("translate", cx, cy)
	drawTarget.Call("rotate", ringTiltAngle)
	drawTarget.Call("translate", -cx, -cy)

	{
		// Clip the lower half of the ellipse to draw only the upper half
		drawTarget.Call("save")
		drawTarget.Call("beginPath")
		drawTarget.Call("rect", cx-outerRingRadius, cy-outerRingRadius, 2*outerRingRadius, outerRingRadius)
		drawTarget.Call("closePath")
		drawTarget.Call("clip")

		// Draw the full ellipse, but only the upper half will be visible due to clipping
		drawTarget.Call("beginPath")
		drawTarget.Call("ellipse", cx, cy, outerRingRadius, innerRingRadius*0.4, 0, 0, 2*math.Pi, false)
		drawTarget.Call("closePath")
		drawTarget.Set("strokeStyle", "rgba(169, 169, 169, 0.8)") // Dark Gray for rings
		drawTarget.Set("lineWidth", radius*0.05)
		drawTarget.Call("stroke")

		// Restore to remove clipping
		drawTarget.Call("restore")
	}

	// Restore context before drawing the planet's body
	drawTarget.Call("restore")

	{
		// Draw Uranus's body
		drawArc(cx, cy, radius, "")

		// Cyan color for Uranus
		gradient := createRadialGradient(cx, cy, radius*0.3, radius, []colorStop{
			{0, "#E0FFFF"},   // Light Cyan at the center
			{0.5, "#AFEEEE"}, // Pale Turquoise
			{1, "#5F9EA0"},   // Cadet Blue at the edges
		})
		drawTarget.Set("fillStyle", gradient)
		drawTarget.Call("fill")

		// Clip to the planet's circle to restrict the gas bands within Uranus's shape
		drawTarget.Call("save") // Save the current drawing state before clipping
		drawTarget.Call("clip")

		// Add gas bands
		bandColors := []string{
			"rgba(176, 224, 230, 0.7)", // #B0E0E6 (Powder Blue) with 70% opacity
			"rgba(173, 216, 230, 0.7)", // #ADD8E6 (Light Blue) with 70% opacity
			"rgba(135, 206, 235, 0.7)", // #87CEEB (Sky Blue) with 70% opacity
			"rgba(135, 206, 250, 0.7)", // #87CEFA (Light Sky Blue) with 70% opacity
		}
		bandHeight := radius * 2 / float64(len(bandColors)) // Height of each band

		for i, color := range bandColors {
			y := cy - radius + float64(i)*bandHeight

			drawTarget.Call("beginPath")
			drawTarget.Call("rect", cx-radius, y, radius*2, bandHeight)
			drawTarget.Set("fillStyle", color)
			drawTarget.Call("fill")
			drawTarget.Call("closePath")
		}

		drawTarget.Call("restore") // Restore the drawing state, removing the clip
	}

	// Save the context and rotate for the ring's tilt
	drawTarget.Call("save")
	drawTarget.Call("translate", cx, cy)
	drawTarget.Call("rotate", ringTiltAngle)
	drawTarget.Call("translate", -cx, -cy)

	{
		// Clip the upper half of the ellipse to draw only the lower half
		drawTarget.Call("save")
		drawTarget.Call("beginPath")
		drawTarget.Call("rect", cx-outerRingRadius, cy, 2*outerRingRadius, outerRingRadius)
		drawTarget.Call("closePath")
		drawTarget.Call("clip")

		// Draw the full ellipse, but only the lower half will be visible due to clipping
		drawTarget.Call("beginPath")
		drawTarget.Call("ellipse", cx, cy, outerRingRadius, innerRingRadius*0.4, 0, 0, 2*math.Pi, false)
		drawTarget.Call("closePath")
		drawTarget.Set("strokeStyle", "rgba(169, 169, 169, 0.8)") // Dark Gray for rings
		drawTarget.Set("lineWidth", radius*0.05)
		drawTarget.Call("stroke")

		// Restore to remove clipping
		drawTarget.Call("restore")
	}

	// Restore the context to remove the rotation
	drawTarget.Call("restore")

	// Reset the line width to the default value
	drawTarget.Set("lineWidth", 1.0)
}

// DrawPlanetVenus is a function that draws Venus on the document.
func DrawPlanetVenus(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	// Draw the main body of Venus with a gradient to simulate the thick atmosphere
	drawArc(cx, cy, radius, "")

	// Enhanced gradient with multiple stops to create depth
	gradient := createRadialGradient(cx, cy, radius*0.2, radius, []colorStop{
		{0, "#FFF8DC"},   // CornSilk at the center for a bright, hazy look
		{0.5, "#F0E68C"}, // Khaki in the middle for a yellowish hue
		{1, "#D2B48C"},   // Tan at the edges for a more defined atmospheric layer
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Clip to the planet's circle to restrict the drawing within Venus
	drawTarget.Call("save") // Save the current drawing state before clipping
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")
	drawTarget.Call("clip")

	// Add some cloud patterns or swirls
	clouds := [][4]float64{
		{cx - radius*0.4, cy - radius*0.4, radius * 0.6, radius * 0.2},
		{cx + radius*0.3, cy + radius*0.2, radius * 0.5, radius * 0.25},
		{cx - radius*0.2, cy + radius*0.35, radius * 0.4, radius * 0.15},
	}

	for _, cloud := range clouds {
		drawTarget.Call("beginPath")
		drawTarget.Call("ellipse", cloud[0], cloud[1], cloud[2], cloud[3], 0, 0, 2*math.Pi, false)
		drawTarget.Call("closePath")

		drawTarget.Set("fillStyle", "rgba(255, 255, 255, 0.2)")
		drawTarget.Call("fill")
	}

	drawTarget.Call("restore") // Restore the drawing state, removing the clip
}

// DrawRect is a function that draws a rectangle on the document.
func DrawRect(coords [2]float64, size [2]float64, color string, cornerRadius float64) {
	x, y := coords[0], coords[1]
	width, height := size[0], size[1]

	if cornerRadius == 0 {
		canvasObjectContext.Set("fillStyle", color)
		canvasObjectContext.Call("fillRect", x, y, width, height)
		return
	}

	canvasObjectContext.Set("fillStyle", color)
	canvasObjectContext.Call("beginPath")
	canvasObjectContext.Call("moveTo", x+cornerRadius, y)
	canvasObjectContext.Call("lineTo", x+width-cornerRadius, y)
	canvasObjectContext.Call("quadraticCurveTo", x+width, y, x+width, y+cornerRadius)
	canvasObjectContext.Call("lineTo", x+width, y+height-cornerRadius)
	canvasObjectContext.Call("quadraticCurveTo", x+width, y+height, x+width-cornerRadius, y+height)
	canvasObjectContext.Call("lineTo", x+cornerRadius, y+height)
	canvasObjectContext.Call("quadraticCurveTo", x, y+height, x, y+height-cornerRadius)
	canvasObjectContext.Call("lineTo", x, y+cornerRadius)
	canvasObjectContext.Call("quadraticCurveTo", x, y, x+cornerRadius, y)
	canvasObjectContext.Call("fill")
}

// DrawSpaceship is a function that draws a spaceship on the document.
// The spaceship is drawn at the specified position (x, y) with the specified width and height.
// The spaceship is drawn facing the specified direction.
// The spaceship is colored with the specified color.
// The spaceship can have a label displayed above or below it.
// The spaceship can have status bars displayed above or below it.
func DrawSpaceship(coors [2]float64, size [2]float64, faceUp bool, color, label string, statusValues []float64, statusColors []string) {
	x, y := coors[0], coors[1]
	width, height := size[0], size[1]

	canvasObjectContext.Set("fillStyle", color)
	canvasObjectContext.Set("strokeStyle", "black")

	// Draw the body of the spaceship
	canvasObjectContext.Call("fillRect", x+width*0.4, y+height*0.2, width*0.2, height*0.6)
	canvasObjectContext.Call("strokeRect", x+width*0.4, y+height*0.2, width*0.2, height*0.6)

	// Draw the wings
	canvasObjectContext.Call("beginPath")
	canvasObjectContext.Call("moveTo", x+width*0.4, y+height*0.2) // Left point of left wing
	if faceUp {
		canvasObjectContext.Call("lineTo", x, y+height*0.75) // Bottom point of left wing
	} else {
		canvasObjectContext.Call("lineTo", x, y+height*0.25) // Bottom point of left wing
	}
	canvasObjectContext.Call("lineTo", x+width*0.4, y+height*0.8) // Right point of left wing
	canvasObjectContext.Call("closePath")
	canvasObjectContext.Call("fill")
	canvasObjectContext.Call("stroke")

	canvasObjectContext.Call("beginPath")
	canvasObjectContext.Call("moveTo", x+width*0.6, y+height*0.2) // Right point of right wing
	if faceUp {
		canvasObjectContext.Call("lineTo", x+width, y+height*0.75) // Bottom point of right wing
	} else {
		canvasObjectContext.Call("lineTo", x+width, y+height*0.25) // Bottom point of right wing
	}
	canvasObjectContext.Call("lineTo", x+width*0.6, y+height*0.8) // Left point of right wing
	canvasObjectContext.Call("closePath")
	canvasObjectContext.Call("fill")
	canvasObjectContext.Call("stroke")

	// Draw the tip of the spaceship
	canvasObjectContext.Call("beginPath")
	if faceUp {
		canvasObjectContext.Call("moveTo", x+width*0.4, y+height*0.2) // Left point of the tip
		canvasObjectContext.Call("lineTo", x+width*0.5, y)            // Top point of the tip
		canvasObjectContext.Call("lineTo", x+width*0.6, y+height*0.2) // Right point of the tip
	} else {
		canvasObjectContext.Call("moveTo", x+width*0.4, y+height*0.8) // Left point of the tip
		canvasObjectContext.Call("lineTo", x+width*0.5, y+height)     // Bottom point of the tip
		canvasObjectContext.Call("lineTo", x+width*0.6, y+height*0.8) // Right point of the tip
	}
	canvasObjectContext.Call("closePath")
	canvasObjectContext.Call("fill")
	canvasObjectContext.Call("stroke")

	// Draw the label above or below the spaceship
	if label != "" {
		canvasObjectContext.Set("font", "16px Arial") // Set font

		// Shorten the label if it is too long.
		// Commandant names come from a browser prompt, so the label can hold
		// multi-byte runes that a byte slice would cut in half.
		if runes := []rune(label); len(runes) > Config.Spaceship.MaximumLabelLength {
			label = string(runes[:Config.Spaceship.MaximumLabelLength-3]) + "..."
		}

		// Measure the width of the label text
		textMetrics := canvasObjectContext.Call("measureText", label)
		labelWidth := textMetrics.Get("width").Float()

		labelX := x + (width-labelWidth)/2 // Center the label horizontally

		var labelY float64
		if faceUp {
			labelY = y + height + 10 // Below the spaceship
		} else {
			labelY = y - 5 // Above the spaceship
		}

		// Draw the label text with a black stroke and fill
		canvasObjectContext.Set("strokeStyle", "black")
		canvasObjectContext.Call("strokeText", label, labelX, labelY)

		canvasObjectContext.Set("fillStyle", color) // Set text color
		canvasObjectContext.Call("fillText", label, labelX, labelY)
	}

	// Draw the status bars
	for i := 0; i < len(statusColors) && i < len(statusValues); i++ {
		canvasObjectContext.Call("beginPath")
		arcRadius := (width+height)/4 + 5 + float64(7*i) // Radius of the status arc

		canvasObjectContext.Set("lineWidth", 5) // Set line width for the status arc

		var startAngle, endAngle float64
		if faceUp {
			startAngle = math.Pi * 1.25 // Start angle (top-left)
			endAngle = math.Pi * 1.75   // End angle (top-right)
			canvasObjectContext.Call("arc", x+width/2, y+height*0.2, arcRadius, startAngle, endAngle, false)
		} else {
			startAngle = math.Pi * 0.25 // Start angle (bottom-left)
			endAngle = math.Pi * 0.75   // End angle (bottom-right)
			canvasObjectContext.Call("arc", x+width/2, y+height*0.8, arcRadius, startAngle, endAngle, false)
		}

		// Draw the background arc (gray)
		canvasObjectContext.Set("strokeStyle", "rgba(128, 128, 128, 0.3)")
		canvasObjectContext.Call("stroke")

		value := statusValues[i]
		if value > 1 {
			value = 1
		}

		actualAngle := startAngle + (endAngle-startAngle)*value

		canvasObjectContext.Call("beginPath")
		if faceUp {
			canvasObjectContext.Call("arc", x+width/2, y+height*0.2, arcRadius, startAngle, actualAngle, false)
		} else {
			canvasObjectContext.Call("arc", x+width/2, y+height*0.8, arcRadius, startAngle, actualAngle, false)
		}

		canvasObjectContext.Set("strokeStyle", statusColors[i])
		canvasObjectContext.Call("stroke")

		canvasObjectContext.Set("lineWidth", 1)
	}
}

// DrawStar draws a star on the invisible canvas to be used as a background on the visible one.
// The star is drawn at the specified position (cx, cy) with the specified number of spikes.
// The outer radius and inner radius of the star are specified.
// The star is filled with the specified color.
func DrawStar(coords [2]float64, spikes int, radius, innerRadius float64, color string, brightness float64) {
	cx, cy := coords[0], coords[1] // Center position

	// Calculate the positions of the star
	var positions [][2]float64
	for i, r := 0, 0.0; i < 2*spikes; i++ {
		if i%2 == 0 {
			r = radius
		} else {
			r = innerRadius
		}

		angle := float64(i) * math.Pi / float64(spikes)
		x := cx + math.Cos(angle)*r
		y := cy + math.Sin(angle)*r
		positions = append(positions, [2]float64{x, y})
	}

	// Draw the star
	// Darken the color based on the brightness
	for _, c := range []string{color, fmt.Sprintf("rgba(0, 0, 0, %.2f)", 1-brightness)} {
		invisibleCtx.Call("beginPath")
		invisibleCtx.Set("fillStyle", c)
		invisibleCtx.Call("moveTo", positions[0][0], positions[0][1])
		for i := 1; i < len(positions); i++ {
			invisibleCtx.Call("lineTo", positions[i][0], positions[i][1])
		}
		invisibleCtx.Call("lineTo", positions[0][0], positions[0][1]) // Close the star
		invisibleCtx.Call("closePath")
		invisibleCtx.Call("fill")
	}
}

// DrawSun is a function that draws the Sun on the document.
func DrawSun(coords [2]float64, radius float64) {
	cx, cy := coords[0], coords[1]

	scale := 1.0 + rand.Float64()/10

	// Create a circular path for the Sun
	drawTarget.Call("beginPath")
	drawTarget.Call("arc", cx, cy, scale*radius, 0, 2*math.Pi, false)
	drawTarget.Call("closePath")

	// Use a radial gradient to represent the Sun's glowing appearance
	gradient := createRadialGradient(cx, cy, scale*radius*0.3, scale*radius, []colorStop{
		{0, "rgba(255, 255, 0, 1)"},
		{0.5, "rgba(255, 165, 0, 0.9)"},
		{0.9, "rgba(255, 165, 0, 0.5)"},
		{1, "rgba(255, 165, 0, 0)"},
	})
	drawTarget.Set("fillStyle", gradient)
	drawTarget.Call("fill")

	// Draw sun flares
	numFlares := rand.Intn(9)
	maxFlareLength := radius * 1.5
	minFlareLength := radius * 1.1
	flareThickness := 2.0

	for i := 0; i < numFlares; i++ {
		// Random angle for each flare
		angle := 2 * math.Pi * rand.Float64()

		// Random length for each flare
		flareLength := minFlareLength + rand.Float64()*(maxFlareLength-minFlareLength)

		// Calculate the end point of the flare
		x := cx + flareLength*math.Cos(angle)
		y := cy + flareLength*math.Sin(angle)

		// Set the style for the flare
		drawTarget.Set("lineWidth", flareThickness)

		gradient := drawTarget.Call("createLinearGradient", cx, cy, x, y)
		gradient.Call("addColorStop", 0, "rgba(255, 255, 0, 1)")     // Bright yellow at the start
		gradient.Call("addColorStop", 0.5, "rgba(255, 255, 0, 0.9)") // Semi-transparent yellow halfway
		gradient.Call("addColorStop", 0.9, "rgba(255, 165, 0, 0.9)") // Semi-transparent orange near the end
		gradient.Call("addColorStop", 1, "rgba(255, 165, 0, 0)")     // Transparent orange at the end

		drawTarget.Set("strokeStyle", gradient)

		// Draw the flare
		drawTarget.Call("beginPath")
		drawTarget.Call("moveTo", cx, cy)
		drawTarget.Call("lineTo", x, y)
		drawTarget.Call("stroke")
		drawTarget.Call("closePath")
	}

	drawTarget.Set("lineWidth", 1.0) // Reset line width
}
