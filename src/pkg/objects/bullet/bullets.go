package bullet

import "github.com/sarumaj/edu-space-invaders/src/pkg/numeric"

// Bullets represents a collection of bullets.
type Bullets []Bullet

// Reload creates a new bullet at the specified position.
// The bullet has the specified damage and skew ratio.
func (bullets *Bullets) Reload(position numeric.Position, damage int, skew, speedBoost numeric.Number) {
	*bullets = append(*bullets, *Craft(position, damage, skew, speedBoost))
}

// Update updates the bullets.
// It moves the bullets and removes the ones that are out of the screen.
// The scale is how far the frame advances the simulation, expressed in nominal
// frames.
func (bullets *Bullets) Update(scale numeric.Number) {
	var visibleBullets []Bullet
	for i := range *bullets {
		bullet := &(*bullets)[i]

		if bullet.Exhausted {
			continue
		}

		bullet.Move(scale)
		if bullet.Position.Y < 0 {
			continue
		}

		visibleBullets = append(visibleBullets, *bullet)
	}

	*bullets = visibleBullets
}
