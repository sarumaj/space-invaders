package bullet

import (
	"fmt"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/enemy"
)

// Bullet represents a bullet shot by the spaceship or by an enemy.
type Bullet struct {
	Position    numeric.Position // Position of the bullet
	Size        numeric.Size     // Size of the bullet
	Speed       numeric.Number   // Speed and damage of the bullet
	Damage      int              // Damage is the amount of health points the bullet takes from the enemy
	Skew        numeric.Number   // Skew of the bullet
	Heading     numeric.Number   // Heading is -1 for a bullet travelling up the screen and +1 for one travelling down
	Exhausted   bool             // Exhausted is true if the bullet is out of the screen or has hit an enemy
	repelVector numeric.Position // Repel vector of the bullet
}

// Hostile reports whether the bullet was fired by an enemy, which is the case
// exactly when it travels down the screen towards the spaceship.
func (bullet Bullet) Hostile() bool { return bullet.Heading > 0 }

// Area returns the area of the bullet.
func (bullet Bullet) Area() numeric.Number {
	if config.Config.Control.CollisionDetectionVersion.Get() == 3 {
		return numeric.GetSkewedLineVertices(bullet.Position, bullet.Size, bullet.Skew).Vertices().Area()
	}
	return bullet.Size.Area()
}

// Draw draws the bullet.
// The bullet is drawn as a yellow rectangle.
func (bullet Bullet) Draw() {
	var color string
	switch {
	case bullet.Hostile():
		// Enemy fire is deliberately off the player's damage ramp: it has to be
		// told apart from the player's own bullets at a glance, not ranked.
		color = "MediumSpringGreen"
	case bullet.Damage > 25_000:
		color = "DarkRed" // Very high damage, intense color
	case bullet.Damage > 12_000:
		color = "Crimson" // High damage, strong but less intense
	case bullet.Damage > 5_000:
		color = "DarkOrange" // Moderate damage, warm color
	case bullet.Damage > 1_000:
		color = "Orange" // Lower damage, vibrant but softer
	case bullet.Damage > 100:
		color = "Gold" // Low damage, bright and shiny
	default:
		color = "Lavender" // Minimal damage, soft and neutral
	}

	// Calculate the end position of the bullet with skew
	horizontalSkew := bullet.Skew * bullet.Size.Height
	// Preserve the total length of the bullet
	verticalComponent := (bullet.Size.Height.Pow(2) - horizontalSkew.Pow(2)).Root()
	// The tail trails behind the bullet, so it points against the heading.
	endPosition := bullet.Position.Add(numeric.Locate(-horizontalSkew, -verticalComponent*bullet.Heading))
	config.DrawLine(
		bullet.Position.Pack(),    // Start position
		endPosition.Pack(),        // End position
		color,                     // Color
		bullet.Size.Width.Float(), // Width
	)
}

// HasHitSpaceship reports whether the bullet is inside the box occupied by the
// spaceship. A bullet is small enough next to the spaceship that the box test is
// indistinguishable from the exact hull test used against enemies, and it keeps
// the target describable without the enemy package.
func (bullet Bullet) HasHitSpaceship(position numeric.Position, size numeric.Size) bool {
	return bullet.Position.X >= position.X &&
		bullet.Position.X <= position.X+size.Width &&
		bullet.Position.Y >= position.Y &&
		bullet.Position.Y <= position.Y+size.Height
}

// Exhaust sets the bullet as exhausted.
func (bullet *Bullet) Exhaust() {
	bullet.Exhausted = true
}

// GetDamage returns the damage of the bullet.
// If the bullet is repelled, the damage is reduced based on the speed of the bullet and the repelling vector.
func (bullet Bullet) GetDamage() int {
	if bullet.repelVector.IsZero() {
		return bullet.Damage
	}

	// Calculate the damage reduction based on the repelling vector
	damageReduction := (bullet.repelVector.Magnitude() / bullet.Speed).Clamp(0, 1)
	return (numeric.Number(bullet.Damage) * damageReduction).Int()
}

// HasHit returns true if the bullet has hit the enemy.
// It uses the Separating Axis Theorem to check for collision.
// The Separating Axis Theorem states that if two convex shapes do not overlap on any axis, then they do not intersect.
// The axes to test are the normals to the edges of the spaceship polygon and the bullet rectangle.
// If there is a separating axis, there is no collision.
// It assumes that the bullet is a rectangle and the enemy is a spaceship polygon.
func (bullet Bullet) HasHit(e enemy.Enemy) bool {
	switch config.Config.Control.CollisionDetectionVersion.Get() {
	case 1:
		return !numeric.GetRectangularVertices(bullet.Position, bullet.Size, false).
			Vertices().
			HasSeparatingAxis(numeric.GetRectangularVertices(e.Geometry.Position(), e.Geometry.Size(), true).
				Vertices())

	case 2:
		return !numeric.GetRectangularVertices(bullet.Position, bullet.Size, false).
			Vertices().
			HasSeparatingAxis(numeric.GetSpaceshipVerticesV1(e.Geometry.Position(), e.Geometry.Size(), e.Type() == enemy.Tank).
				Vertices())

	case 3:
		return !numeric.
			GetSkewedLineVertices(bullet.Position, bullet.Size, bullet.Skew).
			Vertices().
			HasSeparatingAxis(numeric.GetSpaceshipVerticesV2(e.Geometry.Position(), e.Geometry.Size(), e.Type() == enemy.Tank).
				Vertices())

	}

	return false
}

// Move moves the bullet.
// The bullet moves upwards and slightly to the left or right.
// The skew of the bullet is based on the position of the cannon.
// If the bullet is repelled, it moves in the direction of the minimum translation vector.
func (bullet *Bullet) Move(scale numeric.Number) {
	if !bullet.repelVector.IsZero() { // Repel the bullet
		// Apply repelling motion
		bullet.Position = bullet.Position.Add(bullet.repelVector.Mul(scale))

		// Adjust skew based on repelling vector
		bullet.Skew += (bullet.repelVector.X / bullet.Speed)
		bullet.Skew = bullet.Skew.Clamp(-1, 1)

		// Reduce repelling force
		numberOfFrames := numeric.Number(config.Config.Bullet.SpeedDecayDuration.Seconds() *
			config.Config.Control.DesiredFramesPerSecondRate)
		reduction := numeric.E.Pow(-bullet.Speed.Log()*scale/numberOfFrames).Clamp(0, 1)
		bullet.repelVector = bullet.repelVector.Mul(reduction)

		// Stop repelling if the force is too low
		if bullet.repelVector.Magnitude() < 1 {
			bullet.Exhaust()
		}

		return
	}

	bullet.Position = bullet.Position.Add(numeric.Locate(
		bullet.Skew*bullet.Speed*scale,
		bullet.Heading*bullet.Speed*scale,
	))
}

// Repel repels the bullet from the enemy.
func (bullet *Bullet) Repel(e enemy.Enemy) numeric.Position {
	// Calculate the effective area (as substitute for mass)
	bulletArea, enemyArea := bullet.Area(), e.Area()
	effectiveArea := bulletArea * enemyArea / (bulletArea + enemyArea)

	// Calculate the minimum translation vector (MTV)
	var mtv numeric.Position
	switch config.Config.Control.CollisionDetectionVersion.Get() {
	case 1:
		mtv = numeric.GetRectangularVertices(bullet.Position, bullet.Size, false).
			Vertices().
			MinimumTranslationVector(numeric.GetRectangularVertices(e.Geometry.Position(), e.Geometry.Size(), true).
				Vertices())

	case 2:
		mtv = numeric.GetRectangularVertices(bullet.Position, bullet.Size, false).
			Vertices().
			MinimumTranslationVector(numeric.GetSpaceshipVerticesV1(e.Geometry.Position(), e.Geometry.Size(), e.Type() == enemy.Tank).
				Vertices())

	case 3:
		mtv = numeric.GetSkewedLineVertices(bullet.Position, bullet.Size, bullet.Skew).
			Vertices().
			MinimumTranslationVector(numeric.GetSpaceshipVerticesV2(e.Geometry.Position(), e.Geometry.Size(), e.Type() == enemy.Tank).
				Vertices())

	}

	// Apply the MTV to the bullet and the enemy
	bullet.repelVector = mtv.Mul(effectiveArea / bulletArea)
	bullet.Position = bullet.Position.Add(mtv.Mul(effectiveArea / bulletArea))
	return e.Geometry.Position().Sub(mtv.Mul(effectiveArea / enemyArea))
}

// String returns the string representation of the bullet.
func (bullet Bullet) String() string {
	return fmt.Sprintf("Bullet (Pos: %s, Speed: %g, Damage: %d)", bullet.Position, bullet.Speed, bullet.Damage)
}

// Craft creates a new bullet at the specified position.
// The heading is -1 for a bullet fired by the spaceship and +1 for one fired by
// an enemy.
func Craft(position numeric.Position, damage int, skew, speedBoost, heading numeric.Number) *Bullet {
	bullet := Bullet{
		Position: position,
		Size:     numeric.Locate(config.Config.Bullet.Width, config.Config.Bullet.Height).ToBox(),
		Speed:    numeric.Number(config.Config.Bullet.Speed) + speedBoost,
		Damage:   numeric.Randomize(damage, 0.3),
		Skew:     skew,
		Heading:  heading,
	}

	return &bullet
}
