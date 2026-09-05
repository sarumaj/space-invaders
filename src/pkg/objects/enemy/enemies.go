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
	highestProgress := enemies.GetHighestProperty(func(e Enemy) numeric.Number {
		return numeric.Number(e.Level.Progress).Max(1)
	}).Int()
	highestType := EnemyType(enemies.GetHighestProperty(func(e Enemy) numeric.Number {
		return numeric.Number(e.kind)
	}).Int())

	newEnemy := Challenge(name, randomY)
	newEnemy.ToProgressLevel(highestProgress)
	newEnemy.Surprise(Tank, Cloaked, Freezer)
	newEnemy.BerserkGivenAncestor(highestType)

	*enemies = append(*enemies, *newEnemy)
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
	highestType := EnemyType(enemies.GetHighestProperty(func(e Enemy) numeric.Number {
		return numeric.Number(e.kind)
	}).Int())

	var visibleEnemies Enemies
	for i := range *enemies {
		enemy := &(*enemies)[i]
		if enemy.Level.HitPoints <= 0 {
			if *config.Config.Enemy.Regenerate {
				visibleEnemies.AppendNew("", false)
			}

			continue
		}

		enemy.Move(spaceshipPosition, scale)
		canvasDimensions := config.CanvasBoundingBox()
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
