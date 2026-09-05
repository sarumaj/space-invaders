//go:build js && wasm

package config

import (
	"fmt"
	"math"
)

// Destruction animations.
// Destroying an enemy used to set its hit points to zero and play a sound; the
// enemy simply stopped being drawn. Each style below gives the kill a shape that
// matches the hull that died, so the player can see what they just killed.
const (
	BlastBurst     = "burst"     // An expanding ring of shards.
	BlastImplosion = "implosion" // A collapse inwards followed by a flash.
	BlastShatter   = "shatter"   // Crystal splinters flying apart.
	BlastShockwave = "shockwave" // Concentric rings racing outwards.
	BlastSpiral    = "spiral"    // Debris thrown along a spinning arc.
	BlastVaporize  = "vaporize"  // A soft cloud that thins out and drifts.
)

// DrawBlast draws one frame of a destruction animation.
// The progress runs from 0 at the moment of destruction to 1 when the animation
// is spent, and the seed decorrelates two blasts of the same style that happen at
// the same time.
func DrawBlast(coords [2]float64, radius float64, style, color string, progress, seed float64) {
	cx, cy := coords[0], coords[1]
	fade := 1 - progress

	switch style {
	case BlastImplosion:
		// Everything rushes into the centre, then the core lets go.
		collapse := radius * (1 - progress)
		for i := 0; i < 8; i++ {
			angle := seed + float64(i)*math.Pi/4
			drawArc(cx+math.Cos(angle)*collapse, cy+math.Sin(angle)*collapse, radius*0.12*fade,
				fmt.Sprintf("rgba(255, 255, 255, %.2f)", fade))
		}

		if progress > 0.7 {
			flash := (progress - 0.7) / 0.3
			drawArc(cx, cy, radius*flash*1.4, fmt.Sprintf("rgba(200, 200, 255, %.2f)", 1-flash))
		}

	case BlastShatter:
		// Splinters spin away along straight lines, as a crystal would break.
		for i := 0; i < 10; i++ {
			angle := seed + float64(i)*2*math.Pi/10
			distance := radius * progress * 2.2
			length := radius * 0.4 * fade

			startX, startY := cx+math.Cos(angle)*distance, cy+math.Sin(angle)*distance
			drawTarget.Call("beginPath")
			drawTarget.Call("moveTo", startX, startY)
			drawTarget.Call("lineTo", startX+math.Cos(angle)*length, startY+math.Sin(angle)*length)
			drawTarget.Set("strokeStyle", fmt.Sprintf("rgba(220, 240, 255, %.2f)", fade))
			drawTarget.Set("lineWidth", 2)
			drawTarget.Call("stroke")
			drawTarget.Set("lineWidth", 1)
		}

	case BlastShockwave:
		// Concentric rings, the signature of something big coming apart.
		for i := 0; i < 3; i++ {
			offset := progress - float64(i)*0.18
			if offset <= 0 {
				continue
			}

			drawTarget.Call("beginPath")
			drawTarget.Call("arc", cx, cy, radius*offset*3, 0, 2*math.Pi, false)
			drawTarget.Set("strokeStyle", fmt.Sprintf("rgba(255, 210, 120, %.2f)", (1-offset)*0.8))
			drawTarget.Set("lineWidth", 3*(1-offset))
			drawTarget.Call("stroke")
			drawTarget.Set("lineWidth", 1)
		}

		drawArc(cx, cy, radius*fade*0.8, fmt.Sprintf("rgba(255, 160, 60, %.2f)", fade))

	case BlastSpiral:
		// Debris carried around as it flies out, for hulls that were spinning.
		for i := 0; i < 12; i++ {
			angle := seed + float64(i)*2*math.Pi/12 + progress*3
			distance := radius * progress * 2.4
			drawArc(cx+math.Cos(angle)*distance, cy+math.Sin(angle)*distance, radius*0.1*fade,
				fmt.Sprintf("rgba(255, 255, 255, %.2f)", fade*0.9))
		}

	case BlastVaporize:
		// A cloud that thins rather than bursts, for something that was barely
		// there in the first place.
		gradient := createRadialGradient(cx, cy, 0, radius*(1+progress*1.5), []colorStop{
			{0, fmt.Sprintf("rgba(200, 220, 230, %.2f)", fade*0.7)},
			{0.6, fmt.Sprintf("rgba(140, 160, 180, %.2f)", fade*0.35)},
			{1, "rgba(120, 140, 160, 0)"},
		})
		drawTarget.Call("beginPath")
		drawTarget.Call("arc", cx, cy, radius*(1+progress*1.5), 0, 2*math.Pi, false)
		drawTarget.Call("closePath")
		drawTarget.Set("fillStyle", gradient)
		drawTarget.Call("fill")

	default: // BlastBurst
		// The plain kill: a hot core inside a ring of shards.
		drawArc(cx, cy, radius*(0.9-progress*0.6), fmt.Sprintf("rgba(255, 236, 150, %.2f)", fade))

		for i := 0; i < 9; i++ {
			angle := seed + float64(i)*2*math.Pi/9
			distance := radius * progress * 2
			size := radius * 0.18 * fade

			fillPolygon([][2]float64{
				{cx + math.Cos(angle)*distance, cy + math.Sin(angle)*distance - size},
				{cx + math.Cos(angle)*distance + size, cy + math.Sin(angle)*distance + size},
				{cx + math.Cos(angle)*distance - size, cy + math.Sin(angle)*distance + size},
			}, fmt.Sprintf("rgba(255, 140, 40, %.2f)", fade))
		}

		// A last wash of the hull colour, so the kill is attributable.
		drawArc(cx, cy, radius*progress*1.6, withAlpha(color, fade*0.25))
	}
}

// withAlpha re-expresses an "rgba(r, g, b, a)" colour at a different alpha.
// Enemy colours arrive in that form from the colour catalogue, and the blast has
// to fade them out without knowing which one it was handed.
func withAlpha(color string, alpha float64) string {
	r, g, b, _, ok := parseRGBA(color)
	if !ok {
		return fmt.Sprintf("rgba(255, 255, 255, %.2f)", alpha)
	}

	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", r, g, b, alpha)
}
