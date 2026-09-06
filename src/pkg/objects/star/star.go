package star

import (
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// Star represents a star.
type Star struct {
	InnerRadius numeric.Number
	Position    numeric.Position
	Radius      numeric.Number
	Spikes      numeric.Number
	Exhausted   bool
	color       string
}

// Draw is a method that draws the star.
func (star Star) Draw() {
	if star.Exhausted {
		return
	}

	config.DrawStar(
		star.Position.Pack(),
		star.Spikes.Int(),
		star.Radius.Float(),
		star.InnerRadius.Float(),
		star.color,
		config.Config.Star.Brightness,
	)
}

// Exhaust is a method that sets the star as exhausted.
func (star *Star) Exhaust() {
	star.Exhausted = true
}

// Twinkle is a function that creates a new star.
func Twinkle(position numeric.Position) *Star {
	star := Star{
		Position: position,
		Radius:   numeric.RandomRange(config.Config.Star.MinimumRadius, config.Config.Star.MaximumRadius),
		Spikes:   numeric.RandomRange(config.Config.Star.MinimumSpikes, config.Config.Star.MaximumSpikes),
		color: [...]string{
			"White",
			"LightYellow",
			"PaleGoldenrod",
			"LightCyan",
			"SkyBlue",
			"LightSteelBlue",
			"LightSalmon",
			"LightCoral",
			"LightPink",
			"LavenderBlush",
		}[numeric.RandomRange(0, 9).Int()],
	}

	for star.InnerRadius == 0 || star.InnerRadius > star.Radius {
		star.InnerRadius = numeric.RandomRange(config.Config.Star.MinimumInnerRadius, config.Config.Star.MaximumInnerRadius)
	}

	return &star
}
