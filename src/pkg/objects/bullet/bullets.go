package bullet

import (
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// Bullets represents a collection of bullets.
type Bullets []Bullet

// Reload creates a new bullet travelling up the screen at the specified
// position. The bullet has the specified damage and skew ratio.
func (bullets *Bullets) Reload(position numeric.Position, damage int, skew, speedBoost numeric.Number) {
	*bullets = append(*bullets, *Craft(position, damage, skew, speedBoost, -1))
}

// ReloadHostile creates a new bullet travelling down the screen at the specified
// position, as fired by an enemy at the spaceship.
func (bullets *Bullets) ReloadHostile(position numeric.Position, damage int, skew, speedBoost numeric.Number) {
	*bullets = append(*bullets, *Craft(position, damage, skew, speedBoost, +1))
}

// Update updates the bullets.
// It moves the bullets and removes the ones that are out of the screen.
// The scale is how far the frame advances the simulation, expressed in nominal
// frames.
func (bullets *Bullets) Update(scale numeric.Number) {
	canvasDimensions := config.CanvasBoundingBox()

	var visibleBullets []Bullet
	for i := range *bullets {
		bullet := &(*bullets)[i]

		if bullet.Exhausted {
			continue
		}

		bullet.Move(scale)

		// Cull at the edge the bullet is travelling towards.
		if bullet.Position.Y < 0 || bullet.Position.Y.Float() > canvasDimensions.OriginalHeight {
			continue
		}

		visibleBullets = append(visibleBullets, *bullet)
	}

	*bullets = visibleBullets
}
