package effect

import (
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// Blast is one destruction animation playing out at a fixed point in space.
type Blast struct {
	Position numeric.Position // Centre of the blast.
	Radius   numeric.Number   // Radius of the object that died.
	Style    string           // Which animation to play, one of the config.Blast constants.
	Color    string           // Colour of the object that died, so the kill stays attributable.
	Seed     numeric.Number   // Rotates the debris, so two blasts of the same style do not look alike.

	age  numeric.Number // How many nominal frames the blast has been running.
	life numeric.Number // How many nominal frames the blast runs for in total.
}

// Progress returns how far the blast has run, from 0 to 1.
func (blast Blast) Progress() numeric.Number {
	if blast.life <= 0 {
		return 1
	}

	return (blast.age / blast.life).Clamp(0, 1)
}

// Spent reports whether the blast has finished playing.
func (blast Blast) Spent() bool { return blast.Progress() >= 1 }

// Blasts is a collection of destruction animations.
type Blasts []Blast

// Detonate starts a new blast.
func (blasts *Blasts) Detonate(position numeric.Position, radius numeric.Number, style, color string) {
	*blasts = append(*blasts, Blast{
		Position: position,
		Radius:   radius,
		Style:    style,
		Color:    color,
		Seed:     numeric.RandomRange(0, 6.28),
		life: numeric.Number(config.Config.Effect.BlastDuration.Seconds() *
			config.Config.Control.DesiredFramesPerSecondRate),
	})
}

// Draw draws every blast that is still running.
func (blasts Blasts) Draw() {
	for _, blast := range blasts {
		config.DrawBlast(
			blast.Position.Pack(),
			blast.Radius.Float(),
			blast.Style,
			blast.Color,
			blast.Progress().Float(),
			blast.Seed.Float(),
		)
	}
}

// Update ages the blasts and drops the ones that have finished.
// The scale is how far the frame advances the simulation, expressed in nominal
// frames, so a blast lasts the same wall-clock time at any refresh rate.
func (blasts *Blasts) Update(scale numeric.Number) {
	var running Blasts
	for i := range *blasts {
		blast := &(*blasts)[i]
		blast.age += scale

		if blast.Spent() {
			continue
		}

		running = append(running, *blast)
	}

	*blasts = running
}
