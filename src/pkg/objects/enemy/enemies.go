package enemy

import (
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// Enemies represents a collection of enemies.
type Enemies []Enemy

// AppendNew appends a new enemy to the collection.
// The new enemy is created with the specified name and random Y position.
// The new enemy is placed at the highest level of the existing enemies.
// The new enemy is turned into a goodie and berserk based on the probabilities.
func (enemies *Enemies) AppendNew(name string, randomY bool) {
	enemies.AppendScaled(name, randomY, enemies.HighestProgress(), enemies.HighestType())
}

// AppendScaled appends a new enemy scaled against the given progress and
// ancestor type rather than against the collection it is appended to.
// Update rebuilds the fleet into a fresh slice, so an enemy replaced there would
// otherwise be scaled against only the enemies already copied across, making a
// kill early in the list worth a weaker replacement than the same kill late in
// it.
func (enemies *Enemies) AppendScaled(name string, randomY bool, progress int, ancestor EnemyType) {
	newEnemy := Challenge(name, randomY)
	newEnemy.ToProgressLevel(progress)
	newEnemy.Surprise(Tank, Cloaked, Freezer)
	newEnemy.BerserkGivenAncestor(ancestor)

	*enemies = append(*enemies, *newEnemy)
}

// HighestProgress returns the level of the most advanced enemy, at least 1.
func (enemies Enemies) HighestProgress() int {
	return enemies.GetHighestProperty(func(e Enemy) numeric.Number {
		return numeric.Number(e.Level.Progress).Max(1)
	}).Int()
}

// HighestType returns the most dangerous enemy type present.
func (enemies Enemies) HighestType() EnemyType {
	return EnemyType(enemies.GetHighestProperty(func(e Enemy) numeric.Number {
		return numeric.Number(e.kind)
	}).Int())
}

// Count returns the number of enemies of the given type.
func (enemies Enemies) Count(enemyType EnemyType) int {
	var count int
	for _, enemy := range enemies {
		if enemy.kind == enemyType {
			count++
		}
	}

	return count
}

// GetHighestProperty returns the highest value of the property of the enemies.
func (enemies Enemies) GetHighestProperty(property func(Enemy) numeric.Number) numeric.Number {
	var highest numeric.Number
	for _, enemy := range enemies {
		if property(enemy) > highest {
			highest = property(enemy)
		}
	}

	return highest
}

// Update updates the enemies.
// It moves the enemies and removes the ones that are out of the screen
// or have no health points.
// If the regenerate function is provided, it regenerates the enemies.
// The enemies are regenerated when the spaceship reaches the bottom of the screen.
// The new enemies are placed at the highest level of the existing enemies.
// The new enemies are turned into a goodie and berserk based on the probabilities.
func (enemies *Enemies) Update(spaceshipPosition numeric.Position, scale numeric.Number) {
	canvasDimensions := config.CanvasBoundingBox()

	// Measured over the whole fleet before it is rebuilt, so that a replacement
	// is scaled against every enemy alive this frame rather than against however
	// many of them happen to precede it in the slice.
	highestProgress, highestType := enemies.HighestProgress(), enemies.HighestType()

	var visibleEnemies Enemies
	for i := range *enemies {
		enemy := &(*enemies)[i]
		if enemy.Level.HitPoints <= 0 {
			if *config.Config.Enemy.Regenerate {
				visibleEnemies.AppendScaled("", false, highestProgress, highestType)
			}

			continue
		}

		enemy.Move(spaceshipPosition, scale)
		enemy.FadeFlash(scale)
		if enemy.Geometry.Position().Y.Float() >= canvasDimensions.OriginalHeight {
			newEnemy := Challenge(enemy.Name, false)
			newEnemy.ToProgressLevel(enemy.Level.Progress)
			newEnemy.Surprise(Tank, Cloaked, Freezer)
			newEnemy.BerserkGivenAncestor(highestType)
			*enemy = *newEnemy
		}

		visibleEnemies = append(visibleEnemies, *enemy)
	}

	*enemies = visibleEnemies
}
