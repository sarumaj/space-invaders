package enemy

import (
	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/graphics"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

const (
	Normal      EnemyType = iota // Normal is the default enemy type
	Tank                         // Tank is the undestroyable enemy type which can boost the player's spaceship
	Freezer                      // Freezer is the enemy type that can freeze the player's spaceship
	Cloaked                      // Cloaked is the enemy type that is almost invisible to the player
	Berserker                    // Berserker is the enemy type that can harm the player's spaceship more than the normal enemy
	Annihilator                  // Annihilator is the enemy type that can harm the player's spaceship more than the berserker enemy
	Juggernaut                   // Juggernaut is the enemy type that can harm the player's spaceship more than the annihilator enemy
	Dreadnought                  // Dreadnought is the enemy type that can harm the player's spaceship more than the juggernaut enemy
	Behemoth                     // Behemoth is the enemy type that can harm the player's spaceship more than the dreadnought enemy
	Colossus                     // Colossus is the enemy type that can harm the player's spaceship more than the behemoth enemy
	Leviathan                    // Leviathan is the enemy type that can harm the player's spaceship more than the colossus enemy
	Bulwark                      // Bulwark is the enemy type that can harm the player's spaceship more than the leviathan enemy
	Overlord                     // Overlord is the enemy type that can harm the player's spaceship more than the bulwark enemy
)

const (
	Chaser  Behaviour = iota // Chaser homes in on the spaceship.
	Drifter                  // Drifter holds its line and ignores the spaceship.
	Strafer                  // Strafer sweeps sideways as it descends.
	Charger                  // Charger commits to the spaceship's column and keeps closing.
	Lurker                   // Lurker hangs back, edges closer and is hard to see.
)

// Behaviour represents how an enemy type moves.
type Behaviour int

// EnemyType represents the type of the enemy (Normal, Tank, Freezer, Berserker, Annihilator, ...)
type EnemyType int

// Armed reports whether the enemy type shoots back.
// The lighter types do not, so that the opening minutes stay about dodging, and
// the roster escalates into a firefight rather than starting as one.
func (enemyType EnemyType) Armed() bool {
	switch enemyType {
	case Normal, Tank:
		return false

	default:
		return true
	}
}

// GetBehaviour returns the movement behaviour of the enemy type.
func (enemyType EnemyType) GetBehaviour() Behaviour {
	b, ok := map[EnemyType]Behaviour{
		Tank:        Drifter,
		Freezer:     Strafer,
		Cloaked:     Lurker,
		Berserker:   Charger,
		Juggernaut:  Charger,
		Dreadnought: Strafer,
		Behemoth:    Drifter,
		Leviathan:   Strafer,
		Bulwark:     Drifter,
		Overlord:    Charger,
	}[enemyType]

	if !ok {
		return Chaser
	}

	return b
}

// AnyOf returns true if the enemy type is any of the given types.
func (enemyType EnemyType) AnyOf(types ...EnemyType) bool {
	for _, typ := range types {
		if enemyType == typ {
			return true
		}
	}

	return false
}

// GetColor returns the color of the enemy based on its type.
func (enemyType EnemyType) GetColor() graphics.Color {
	c, ok := map[EnemyType]graphics.Color{
		Tank:        graphics.Catalogue().Chartreuse(),
		Freezer:     graphics.Catalogue().DeepSkyBlue(),
		Cloaked:     graphics.Catalogue().DarkSlateGray().SetA(0.6),
		Berserker:   graphics.Catalogue().Crimson(),
		Annihilator: graphics.Catalogue().MidnightBlue(),
		Juggernaut:  graphics.Catalogue().DarkOrange(),
		Dreadnought: graphics.Catalogue().DarkRed(),
		Behemoth:    graphics.Catalogue().DarkGreen(),
		Colossus:    graphics.Catalogue().DarkBlue(),
		Leviathan:   graphics.Catalogue().DarkMagenta(),
		Bulwark:     graphics.Catalogue().DarkCyan(),
		Overlord:    graphics.Catalogue().DarkGoldenRod(),
	}[enemyType]

	if !ok {
		return graphics.Catalogue().DarkSeaGreen()
	}

	return c
}

// GetDefenseBoost returns the defense boost of the enemy based on its type.
func (enemyType EnemyType) GetDefenseBoost() int {
	b, ok := map[EnemyType]int{
		Berserker:   config.Config.Enemy.Berserker.DefenseBoost,
		Annihilator: config.Config.Enemy.Annihilator.DefenseBoost,
		Juggernaut:  config.Config.Enemy.Juggernaut.DefenseBoost,
		Dreadnought: config.Config.Enemy.Dreadnought.DefenseBoost,
		Behemoth:    config.Config.Enemy.Behemoth.DefenseBoost,
		Colossus:    config.Config.Enemy.Colossus.DefenseBoost,
		Leviathan:   config.Config.Enemy.Leviathan.DefenseBoost,
		Bulwark:     config.Config.Enemy.Bulwark.DefenseBoost,
		Overlord:    config.Config.Enemy.Overlord.DefenseBoost,
	}[enemyType]

	if !ok {
		return 0
	}

	return b
}

// GetHitpointsBoost returns the hitpoints boost of the enemy based on its type.
func (enemyType EnemyType) GetHitpointsBoost() int {
	b, ok := map[EnemyType]int{
		Berserker:   config.Config.Enemy.Berserker.HitpointsBoost,
		Annihilator: config.Config.Enemy.Annihilator.HitpointsBoost,
		Juggernaut:  config.Config.Enemy.Juggernaut.HitpointsBoost,
		Dreadnought: config.Config.Enemy.Dreadnought.HitpointsBoost,
		Behemoth:    config.Config.Enemy.Behemoth.HitpointsBoost,
		Colossus:    config.Config.Enemy.Colossus.HitpointsBoost,
		Leviathan:   config.Config.Enemy.Leviathan.HitpointsBoost,
		Bulwark:     config.Config.Enemy.Bulwark.HitpointsBoost,
		Overlord:    config.Config.Enemy.Overlord.HitpointsBoost,
	}[enemyType]

	if !ok {
		return 0
	}

	return b
}

// GetScale returns the scale of the enemy based on its type.
func (enemyType EnemyType) GetScale() numeric.Number {
	s, ok := map[EnemyType]numeric.Number{
		Berserker:   numeric.Number(config.Config.Enemy.Berserker.SizeFactorBoost),
		Annihilator: numeric.Number(config.Config.Enemy.Annihilator.SizeFactorBoost),
		Juggernaut:  numeric.Number(config.Config.Enemy.Juggernaut.SizeFactorBoost),
		Dreadnought: numeric.Number(config.Config.Enemy.Dreadnought.SizeFactorBoost),
		Behemoth:    numeric.Number(config.Config.Enemy.Behemoth.SizeFactorBoost),
		Colossus:    numeric.Number(config.Config.Enemy.Colossus.SizeFactorBoost),
		Leviathan:   numeric.Number(config.Config.Enemy.Leviathan.SizeFactorBoost),
		Bulwark:     numeric.Number(config.Config.Enemy.Bulwark.SizeFactorBoost),
		Overlord:    numeric.Number(config.Config.Enemy.Overlord.SizeFactorBoost),
	}[enemyType]

	if !ok {
		return numeric.Number(1)
	}

	return s
}

// GetSpeedFactor returns the speed factor of the enemy based on its type.
func (enemyType EnemyType) GetSpeedFactor() numeric.Number {
	s, ok := map[EnemyType]numeric.Number{
		Cloaked:     numeric.Number(config.Config.Enemy.Cloaked.SpeedModifier),
		Berserker:   numeric.Number(config.Config.Enemy.Berserker.SpeedModifier),
		Annihilator: numeric.Number(config.Config.Enemy.Annihilator.SpeedModifier),
		Juggernaut:  numeric.Number(config.Config.Enemy.Juggernaut.SpeedModifier),
		Dreadnought: numeric.Number(config.Config.Enemy.Dreadnought.SpeedModifier),
		Behemoth:    numeric.Number(config.Config.Enemy.Behemoth.SpeedModifier),
		Colossus:    numeric.Number(config.Config.Enemy.Colossus.SpeedModifier),
		Leviathan:   numeric.Number(config.Config.Enemy.Leviathan.SpeedModifier),
		Bulwark:     numeric.Number(config.Config.Enemy.Bulwark.SpeedModifier),
		Overlord:    numeric.Number(config.Config.Enemy.Overlord.SpeedModifier),
	}[enemyType]

	if !ok {
		return numeric.Number(1)
	}

	return s
}

// GetPenalty returns the penalty of the enemy based on its type.
func (enemyType EnemyType) GetPenalty() int {
	p, ok := map[EnemyType]int{
		Cloaked:     config.Config.Enemy.Cloaked.Penalty,
		Freezer:     config.Config.Enemy.Freezer.Penalty,
		Normal:      config.Config.Enemy.DefaultPenalty,
		Berserker:   config.Config.Enemy.Berserker.Penalty,
		Annihilator: config.Config.Enemy.Annihilator.Penalty,
		Juggernaut:  config.Config.Enemy.Juggernaut.Penalty,
		Dreadnought: config.Config.Enemy.Dreadnought.Penalty,
		Behemoth:    config.Config.Enemy.Behemoth.Penalty,
		Colossus:    config.Config.Enemy.Colossus.Penalty,
		Leviathan:   config.Config.Enemy.Leviathan.Penalty,
		Bulwark:     config.Config.Enemy.Bulwark.Penalty,
		Overlord:    config.Config.Enemy.Overlord.Penalty,
	}[enemyType]

	if !ok {
		return 0
	}

	return p
}

// InRange returns true if the enemy type is in the range of the given types.
func (enemyType EnemyType) InRange(min, max EnemyType) bool {
	if min > max {
		min, max = max, min
	}
	return min <= enemyType && enemyType <= max
}

// Next returns the next enemy type.
func (enemyType EnemyType) Next() EnemyType {
	switch enemyType {
	case Overlord: // Overlord is the last enemy type
		return Overlord
	case Tank: // Tank stays the same
		return enemyType
	case Normal, Freezer, Cloaked: // Normal, Freezer, Cloaked berserk into Berserker
		return Berserker
	default: // The other enemy types berserk into the next enemy type
		return enemyType + 1
	}
}

// String returns the string representation of the enemy type.
func (enemyType EnemyType) String() string {
	return [...]string{
		"Normal",
		"Tank",
		"Freezer",
		"Cloaked",
		"Berserker",
		"Annihilator",
		"Juggernaut",
		"Dreadnought",
		"Behemoth",
		"Colossus",
		"Leviathan",
		"Bulwark",
		"Overlord",
	}[enemyType]
}
