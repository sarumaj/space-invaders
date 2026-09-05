package spaceship

import (
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/enemy"
)

// SpaceshipLevel represents the spaceship level.
type SpaceshipLevel struct {
	AccelerateRate numeric.Number // AccelerateRate is the rate at which the spaceship accelerates.
	Cannons        int            // Cannons is the number of cannons the spaceship has.
	Experience     int            // Experience is the experience level of the spaceship.
	HighScore      int            // HighScore is the high score of the spaceship.
	Progress       int            // Progress is the progress level of the spaceship.
	Shield         *Shield        // Shield is the shield of the spaceship.
}

// Down decreases the spaceship level.
// It uses the shield if it is available.
// If the spaceship level is already at the minimum level, it does nothing.
// If the number of cannons is greater than 1, it decreases the number of cannons by 1.
// It decreases the accelerate rate by the acceleration value: x = -0.5 + sqrt(0.25 + x').
// It returns true if the spaceship level has decreased or if the shield has been used.
func (lvl *SpaceshipLevel) Down() bool {
	switch {
	case lvl.Shield.Use():
		return true

	case config.Config.Control.GodMode.Get():
		return false

	case lvl.Progress == 0:
		return false

	}

	lvl.AccelerateRate = (-0.5 + (0.25 + lvl.AccelerateRate).Root()).
		Max(numeric.Number(config.Config.Spaceship.Acceleration))

	if lvl.Cannons > 1 && (lvl.Progress-1)%config.Config.Spaceship.CannonProgress == 0 {
		lvl.Cannons -= 1
	}

	lvl.Progress -= 1
	lvl.Shield.Reduce()
	return true
}

// GainExperience gains experience for the spaceship.
// It calculates the experience gain based on the enemy type and level using the penalty values.
// If the experience is greater than the required experience, it increases the spaceship level.
// The formula for the required experience is the logarithm of the spaceship progress multiplied by the experience factor.
// It returns true if the spaceship level has increased.
func (lvl *SpaceshipLevel) GainExperience(e enemy.Enemy) bool {
	// Calculate the base experience using the penalty values
	base := map[enemy.EnemyType]numeric.Number{
		enemy.Freezer:     numeric.Number(config.Config.Enemy.Freezer.Penalty),
		enemy.Normal:      numeric.Number(config.Config.Enemy.DefaultPenalty),
		enemy.Berserker:   numeric.Number(config.Config.Enemy.Berserker.Penalty),
		enemy.Annihilator: numeric.Number(config.Config.Enemy.Annihilator.Penalty),
	}[e.Type()]

	// Calculate the experience gain
	gain := (base * numeric.Number(e.Level.Progress)).Int()

	// Increase the experience
	lvl.Experience += gain

	// Check later if the spaceship level has increased
	currentLvl := lvl.Progress

	// Increase the spaceship level
	for required := lvl.GetRequiredExperience(); lvl.Experience >= required; required = lvl.GetRequiredExperience() {
		lvl.Up()
		lvl.Experience -= required // Subtract the required experience
	}

	if lvl.Experience < 0 {
		lvl.Experience = 0
	}

	// Return true if the spaceship level has increased
	return lvl.Progress > currentLvl
}

// ExperienceRatio returns how far the spaceship is towards its next level, from
// 0 to 1.
func (lvl SpaceshipLevel) ExperienceRatio() float64 {
	required := lvl.GetRequiredExperience()
	if required <= 0 {
		return 0
	}

	return numeric.Number(float64(lvl.Experience)/float64(required)).Clamp(0, 1).Float()
}

// GetRequiredExperience returns the required experience for the spaceship.
func (lvl SpaceshipLevel) GetRequiredExperience() int {
	return (numeric.E.Pow(numeric.Number(lvl.Progress) / numeric.Number(config.Config.Spaceship.ExperienceScaler))).Int()
}

// Up increases the spaceship level.
// If the progress is a multiple of the cannon progress, it increases the number of cannons by 1.
// If the number of cannons is less than the maximum number of cannons, it increases the number of cannons by 1.
// It increases the accelerate rate by the acceleration value: x' = x*(1+x).
func (lvl *SpaceshipLevel) Up() {
	if lvl.Cannons < config.Config.Spaceship.MaximumCannons && (lvl.Progress+1)%config.Config.Spaceship.CannonProgress == 0 {
		lvl.Cannons += 1
	}

	lvl.AccelerateRate = (lvl.AccelerateRate * numeric.Number(1+config.Config.Spaceship.Acceleration)).
		Min(numeric.Number(config.Config.Spaceship.MaximumSpeed * config.Config.Spaceship.Acceleration))

	lvl.Progress += 1

	lvl.HighScore = numeric.Number(lvl.HighScore).Max(numeric.Number(lvl.Progress)).Int()

	lvl.Shield.Reinforce()
}
