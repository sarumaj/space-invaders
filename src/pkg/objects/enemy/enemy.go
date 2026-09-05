package enemy

import (
	"fmt"

	"github.com/Pallinder/go-randomdata"
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/graphics"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// Enemy represents the enemy.
type Enemy struct {
	Name                string                    // Name is the name of the enemy.
	Color               *graphics.ColorTransition // Color is the color transition of the enemy.
	Geometry            *graphics.SizeTransition  // Geometry is the size transition of the enemy.
	SpecialtyLikeliness numeric.Number            // SpecialtyLikeliness is the likelihood of the enemy being a tank or a freezer (expected to be lower than 1).
	Level               *EnemyLevel               // Level is the level of the enemy.
	kind                EnemyType                 // Type is the type of the enemy.
}

// Area returns the area of the enemy.
func (enemy Enemy) Area() numeric.Number {
	switch config.Config.Control.CollisionDetectionVersion.Get() {
	case 1:
		return numeric.GetRectangularVertices(enemy.Geometry.Position(), enemy.Geometry.Size(), false).Vertices().Area()
	case 2:
		return numeric.GetSpaceshipVerticesV1(enemy.Geometry.Position(), enemy.Geometry.Size(), enemy.kind == Tank).Vertices().Area()
	case 3:
		return numeric.GetSpaceshipVerticesV2(enemy.Geometry.Position(), enemy.Geometry.Size(), enemy.kind == Tank).Vertices().Area()
	}
	return enemy.Geometry.Size().Area()
}

// Berserk turns the enemy into a berserker or an annihilator.
// If the enemy is a tank or a freezer, it does nothing.
// If the enemy is a normal enemy, it has a chance to become a berserker.
// If the enemy is a berserker, it has a chance to become an annihilator.
// If the enemy is an annihilator, it increases its size, health points and defense.
func (enemy *Enemy) Berserk() {
	if !numeric.SampleUniform(enemy.Level.BerserkLikeliness) {
		return
	}

	enemy.ChangeType(enemy.kind.Next())
	enemy.Level.Up()
}

// BerserkGivenAncestor increases the chance of the enemy to become a berserker or an annihilator
// by repeating the berserk for the new enemy given the enemy type of the ancestor.
func (enemy *Enemy) BerserkGivenAncestor(oldType EnemyType) {
	enemy.Berserk()
	switch oldType {

	case Overlord, Bulwark, Leviathan, Colossus, Behemoth, Dreadnought, Juggernaut, Annihilator, Berserker:
		for i := oldType; i >= Berserker; i-- {
			enemy.Berserk()
		}

	default:
		enemy.Berserk()

	}
}

// ChangeType changes the type of the enemy.
func (enemy *Enemy) ChangeType(newType EnemyType) {
	amplifier := numeric.Number(1)
	if enemy.kind == newType {
		amplifier = numeric.Number(config.Config.Enemy.YetAgainAmplifier)
	}

	enemy.Level.HitPoints += (numeric.Number(newType.GetHitpointsBoost()) * amplifier).Int()
	enemy.Level.Defense += (numeric.Number(newType.GetDefenseBoost()) * amplifier).Int()
	enemy.Level.Speed *= (newType.GetSpeedFactor() * amplifier)

	enemy.Geometry.SetScale(newType.GetScale())
	enemy.Color.SetColor(newType.GetColor())

	enemy.kind = newType
}

// Destroy destroys the enemy.
// The health points of the enemy are set to 0.
func (enemy *Enemy) Destroy() {
	enemy.Level.HitPoints = 0

	go config.PlayAudio("enemy_destroyed.wav", false)
}

// Draw draws the enemy.
// The enemy is drawn as a rectangle with the specified color.
// The color is based on the type of the enemy.
// If the control to draw object labels is enabled, the name of the enemy is drawn.
// If the control to draw enemy hitpoint bars is enabled, the hitpoint bar is drawn.
func (enemy *Enemy) Draw() {
	var label string
	if config.Config.Control.DrawObjectLabels.Get() {
		label = enemy.Name
	}

	var statusValues []float64
	var statusColors []string
	if enemy.kind != Tank && config.Config.Control.DrawEnemyHitpointBars.Get() {
		statusValues = append(statusValues, float64(enemy.Level.HitPoints)/float64(enemy.Level.HitPoints+enemy.Level.HitPointsLoss))
		statusColors = append(statusColors, "rgba(240, 0, 0, 0.8)")
	}

	enemy.Color.Interpolate()
	enemy.Geometry.Interpolate()

	config.DrawSpaceship(
		enemy.Geometry.Position().Pack(),
		enemy.Geometry.Size().Pack(),
		enemy.kind == Tank, // Face up if the enemy is a tank
		enemy.kind.GetColor().FormatRGBA(),
		label,
		statusValues,
		statusColors,
	)
}

// Hit reduces the health points of the enemy.
// The damage is reduced by the defense of the enemy.
// If the damage is less than 0, it is set to 0.
func (enemy *Enemy) Hit(damage int) int {
	damage = damage - enemy.Level.Defense - numeric.RandomRange(0, enemy.Level.Defense*enemy.Level.Progress).Int()
	damage = numeric.Number(damage).Clamp(0, numeric.Number(enemy.Level.HitPoints)).Int()

	enemy.Level.HitPointsLoss += damage
	enemy.Level.HitPoints -= damage

	go config.PlayAudio("enemy_hit.wav", false)

	return damage
}

// IsDestroyed returns true if the enemy is destroyed.
func (enemy Enemy) IsDestroyed() bool { return enemy.Level.HitPoints <= 0 }

// Move moves the enemy.
// The enemy moves downwards and changes its horizontal direction.
// If the enemy is a tank, it moves only downwards and does not change its horizontal direction.
// The direction of the enemy is based on the position of the spaceship.
// If the spaceship is below the enemy, the enemy moves towards the spaceship.
// Otherwise, the enemy moves randomly.
func (enemy *Enemy) Move(spaceshipPosition numeric.Position, scale numeric.Number) {
	if enemy.kind == Tank {
		enemy.Geometry.SetPosition(enemy.Geometry.Position().Add(numeric.Locate(0, numeric.Number(enemy.Level.Speed)*scale)))
		return
	}

	// Calculate the horizontal and vertical distances to the spaceship
	delta := spaceshipPosition.Sub(enemy.Geometry.Position())

	// Calculate the distance to the spaceship
	distance := delta.Magnitude()

	// Define the strength formula
	strength := numeric.Number(enemy.Level.Progress) / (distance + 1) // Add 1 to avoid division by zero

	// Add randomness to the chase based on strength
	delta = delta.Add(numeric.Locate(
		numeric.RandomRange(-0.5, 0.5), // Random number between -0.5 and 0.5
		numeric.RandomRange(-1, 0),     // Random number between -1 and 0
	)).Mul(strength * scale)

	// Limit the speed of the enemy
	if maximum := numeric.Number(config.Config.Enemy.MaximumSpeed) * scale; delta.Magnitude() > maximum {
		delta = delta.Normalize().Mul(maximum)
	}

	// Move down using the speed
	enemy.Geometry.SetPosition(enemy.Geometry.Position().Add(numeric.Locate(0, numeric.Number(enemy.Level.Speed)*scale)))

	// Dash off screen if the spaceship is below the enemy and the enemy is close to the spaceship
	if enemy.Geometry.Position().Y > spaceshipPosition.Y && enemy.Geometry.Position().Sub(spaceshipPosition).X.Abs() < enemy.Geometry.Size().ToVector().Magnitude() {
		delta.Y = numeric.Number(config.Config.Enemy.MaximumSpeed) * scale
	}

	// Move horizontally and vertically towards the spaceship
	enemy.Geometry.SetPosition(enemy.Geometry.Position().Add(delta))
}

// String returns the string representation of the enemy.
func (enemy Enemy) String() string {
	return fmt.Sprintf("%s (Lvl: %d, Pos: %s, HP: %d, Type: %s)", enemy.Name, enemy.Level.Progress, enemy.Geometry.Position(), enemy.Level.HitPoints, enemy.kind)
}

// Surprise turns the enemy into a freezer or a tank.
// If the enemy is a normal enemy, it has a chance to become a freezer, tank or cloaked.
// The likelihood is based on the SpecialtyLikeliness.
func (enemy *Enemy) Surprise(types ...EnemyType) {
	if len(types) == 0 {
		types = append(types, Tank, Freezer, Cloaked)
	}

	var valid []EnemyType
	for _, t := range types {
		switch t {
		case Tank, Freezer, Cloaked:
			valid = append(valid, t)
		}
	}

	if len(valid) > 1 {
		valid = numeric.RandomSort(valid)
	}

	if enemy.kind == Normal && numeric.SampleUniform(enemy.SpecialtyLikeliness) {
		enemy.ChangeType(valid[numeric.RandomRange(0, len(valid)-1).Int()])
	}
}

// ToProgressLevel sets the progress level of the enemy (up or down).
func (enemy *Enemy) ToProgressLevel(progress int) {
	if progress < 1 {
		progress = 1
	}

	for enemy.Level.Progress < progress {
		enemy.Level.Up()
	}

	for enemy.Level.Progress > progress {
		enemy.Level.Down()
	}
}

// Type returns the type of the enemy.
func (enemy Enemy) Type() EnemyType { return enemy.kind }

// Challenge creates a new enemy.
// If the name is empty, a random name is generated.
// If randomY is true, the enemy is placed at a random Y position
// in the top half of the canvas.
// Otherwise, the enemy is placed at the top of the canvas.
// The enemy is a normal enemy.
// The enemy has the initial speed, hit points and defense.
// The likelihood of the enemy becoming a tank is based on the TankLikeliness.
// The likelihood of the enemy becoming a berserker is based on the BerserkLikeliness.
// The enemy has the initial level.
func Challenge(name string, randomY bool) *Enemy {
	if name == "" {
		name = randomdata.SillyName()
	}

	canvasDimensions := config.CanvasBoundingBox()
	var y numeric.Number
	if randomY {
		y = numeric.RandomRange(0, canvasDimensions.OriginalHeight/2)
	}

	enemy := Enemy{
		Color: graphics.InitialColorTransition(Normal.GetColor()),
		Geometry: graphics.InitialSizeTransition(
			numeric.Locate(config.Config.Enemy.Width, config.Config.Enemy.Height).ToBox(),
			numeric.Locate(numeric.RandomRange(0, canvasDimensions.OriginalWidth), y),
		),
		SpecialtyLikeliness: numeric.Number(config.Config.Enemy.SpecialtyLikeliness),
		Level: &EnemyLevel{
			Progress:          1,
			Speed:             numeric.Number(config.Config.Enemy.InitialSpeed),
			HitPoints:         numeric.Randomize(config.Config.Enemy.InitialHitpoints, 0.3),
			Defense:           numeric.Randomize(config.Config.Enemy.InitialDefense, 0.3),
			BerserkLikeliness: numeric.Number(config.Config.Enemy.BerserkLikeliness),
		},
		Name: name,
	}

	return &enemy
}
