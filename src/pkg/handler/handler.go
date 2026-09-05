package handler

import (
	"context"
	"sync"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/bullet"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/enemy"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/planet"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/spaceship"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/star"
)

// handler is the game handler.
type handler struct {
	ctx          context.Context      // ctx is an abortable context of the handler
	cancel       context.CancelFunc   // cancel is the cancel function of the handler
	enemies      enemy.Enemies        // enemies is the list of enemies
	enemyBullets bullet.Bullets       // enemyBullets are the bullets fired by the enemies; they live here because the bullet package already depends on the enemy package
	keyEvent     chan keyEvent        // keyupEvent is the channel for key events
	keysHeld     map[action]bool      // keysHeld is the set of actions whose key is currently down
	mouseEvent   chan mouseEvent      // mouseEvent is the channel for mouse events
	mouseHeld    map[mouseButton]bool // mouseHeld is the map of mouse buttons held
	once         sync.Once            // once is meant to register the keydown event only once
	planet       *planet.Planet       // planet is the planet to be drawn
	spaceship    *spaceship.Spaceship // spaceship is the player's spaceship
	stars        star.Stars           // stars is the list of stars
	touchEvent   chan touchEvent      // touchEvent is the channel for touch events
	touchHeld    bool                 // touchHeld is the flag to indicate if the touch is held
}

// applyGravityOnEnemies applies gravity to the enemies.
// It applies gravity to the enemies, each enemy trapped in the planet's gravity is increasing the planet's mass.
// If the planet is a black hole, it pulls the enemies away, if the spaceship is not within the range of the planet.
func (h *handler) applyGravityOnEnemies() {
	// Apply gravity to the enemies, each enemy trapped in the planet's gravity is increasing the planet's mass.
	// If the planet is a black hole, it repels the enemies away, if the spaceship is not within the range of the planet,
	// to mimic some kind of a intelligent behavior (as if the enemies were trying to avoid the black hole).

	// If the planet is a black hole.
	repel := h.planet.Type == planet.BlackHole
	spaceshipPosition := h.spaceship.Geometry.Position().Add(h.spaceship.Geometry.Size().Half().ToVector())

	for i, e := range h.enemies {
		// And if the spaceship is not within the range of the planet or the enemy is a goodie:
		repel = repel && (!h.planet.WithinRange(spaceshipPosition, 1) || e.Type() == enemy.Tank)
		// And if the enemy is not within the range of the planet:
		repel = repel && !h.planet.WithinRange(e.Geometry.Position().Add(e.Geometry.Size().Half().ToVector()), 1)

		h.enemies[i].Geometry.SetPosition(h.planet.ApplyGravity(
			e.Geometry.Position().Add(e.Geometry.Size().Half().ToVector()),
			e.Area(),
			true,  // Increase the planet's mass
			repel, // Repel the enemies away or not
		).Sub(e.Geometry.Size().Half().ToVector()))

		// Shrink the enemy if it is within range of the black hole.
		if h.planet.Type == planet.BlackHole && h.planet.WithinRange(h.enemies[i].Geometry.Position().Add(e.Geometry.Size().Half().ToVector()), 1) {
			area := e.Area()
			if area <= 1 { // Destroy the enemy if it is too small.
				h.enemies[i].Destroy()
				config.SendMessage(config.Execute(config.Config.MessageBox.Messages.EnemyDestroyed, config.Template{
					"EnemyName": e.Name,
					"EnemyType": e.Type(),
				}), false, true)
				continue
			}

			// Shrink down the enemy.
			h.enemies[i].Geometry.SetAnimationDuration(config.Config.Planet.Impact.BlackHole.ObjectSizeDecayDuration).SetScale(1e-9)

			continue
		}

		// If the enemy has been shrunk, restore the enemy to its original size.
		if h.planet.Type != planet.BlackHole || !h.planet.WithinRange(h.enemies[i].Geometry.Position().Add(e.Geometry.Size().Half().ToVector()), 1.2) {
			h.enemies[i].Geometry.SetAnimationDuration(config.Config.Control.AnimationDuration).SetScale(e.Type().GetScale())
		}
	}
}

// applyGravityOnSpaceship applies gravity to the spaceship.
// It applies gravity to the spaceship.
// The spaceship's mass should not increase the planet's mass.
// If the planet is a black hole or a supernova, it applies gravity to the bullets.
func (h *handler) applyGravityOnSpaceship() {
	// Apply gravity to the spaceship.
	// The spaceship's mass should not increase the planet's mass.
	h.spaceship.Geometry.SetPosition(h.planet.ApplyGravity(
		h.spaceship.Geometry.Position().Add(h.spaceship.Geometry.Size().Half().ToVector()),
		h.spaceship.Area(),
		false, // Do not increase the planet's mass
		false, // Do not reverse the gravity
	).Sub(h.spaceship.Geometry.Size().Half().ToVector()))

	// Correct the spaceship's position if it is out of the canvas.
	h.spaceship.FixPosition()

	if h.planet.Type.AnyOf(planet.BlackHole, planet.Supernova) {
		// Apply gravity to the bullets.
		for i, bullet := range h.spaceship.Bullets {
			h.spaceship.Bullets[i].Position = h.planet.ApplyGravity(
				bullet.Position,
				numeric.Number(config.Config.Bullet.Weight)*bullet.Area(),
				false, // Do not increase the planet's mass
				false, // Do not reverse the gravity
			)

			// Calculate skew based on the angle of the velocity vector
			if delta := h.spaceship.Bullets[i].Position.Sub(bullet.Position); !delta.IsZero() {
				// make the skew proportional to the angle of the velocity vector
				// and the distance from the planet
				ratio := delta.X / delta.Magnitude() * h.planet.Radius / numeric.Number(config.Config.Planet.MaximumRadius)
				proximity := delta.Distance(bullet.Position)
				// apply inverse decay function
				h.spaceship.Bullets[i].Skew = (bullet.Skew + ratio/proximity).Clamp(-1, 1)
			}

			// Exhaust the bullet if it is stuck in the planet.
			if h.planet.WithinRange(h.spaceship.Bullets[i].Position, 0.1) {
				h.spaceship.Bullets[i].Exhaust()
			}
		}
	}

	// If the spaceship is within range of the black hole, shrink the spaceship.
	if h.planet.Type == planet.BlackHole && h.planet.WithinRange(h.spaceship.Geometry.Position().Add(h.spaceship.Geometry.Size().Half().ToVector()), 1) {
		area := h.spaceship.Area()
		// Destroy the spaceship if it is too small.
		if area <= 1 && !config.Config.Control.GodMode.Get() {
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.GameOver, config.Template{
				"DiscoveredPlanets": h.spaceship.Discovered(),
				"HighScore":         h.spaceship.Level.HighScore,
				"Rank":              config.SetScore(h.spaceship.Commandant, h.spaceship.Level.HighScore),
				"Reason":            config.Execute(config.Config.Planet.Impact.BlackHole.SpaceshipDestroyedReason),
				"TopScores":         config.GetScores(10),
			}), false, false)
			h.pause()
			h.cancel()
			return
		}

		if area > 1 { // Shrink down the spaceship.
			h.spaceship.Geometry.
				SetAnimationDuration(config.Config.Planet.Impact.BlackHole.ObjectSizeDecayDuration).
				SetScale(1e-9)
		}

		return
	}

	// If the spaceship has been shrunk, restore the spaceship to its original size.
	if h.planet.Type != planet.BlackHole || !h.planet.WithinRange(h.spaceship.Geometry.Position().Add(h.spaceship.Geometry.Size().Half().ToVector()), 1.2) {
		h.spaceship.Geometry.
			SetAnimationDuration(config.Config.Control.AnimationDuration).
			SetScale(h.spaceship.State().GetScale())
	}
}

// applyPlanetImpact applies the impact of the planet, anomaly, or the sun on the game objects.
// If the planet is Uranus, Neptune, or Pluto, it increases the specialty likeliness of the enemies for freezers.
// If the planet is Mercury, Mars, or Pluto, it increases the berserk likeliness of the enemies.
// If the planet is Jupiter or Saturn, it increases the defense and hitpoints of the enemies.
// If the planet is Venus or Earth, it slows down the spaceship and increases the specialty likeliness of the enemies for goodies.
// If the spaceship is within range of the sun, it unfreezes the spaceship.
// If a freezer is within range of the sun, it unfreezes the freezer.
// If the anomaly is a black hole, it sucks in the bullets and other objects.
// If the anomaly is a supernova, it distorts the bullets and other objects and disables the freezers.
func (h *handler) applyPlanetImpact() {
	defer h.applyGravityOnEnemies()
	defer h.applyGravityOnSpaceship()

	message := config.Execute(
		config.Config.MessageBox.Messages.PlanetImpactsSystem,
		config.Template{
			"PlanetName": h.planet.Type.String(),
			"Description": config.Execute(map[planet.PlanetType]config.TemplateString{
				planet.Mercury:   config.Config.Planet.Impact.Mercury.Description,
				planet.Venus:     config.Config.Planet.Impact.Venus.Description,
				planet.Earth:     config.Config.Planet.Impact.Earth.Description,
				planet.Mars:      config.Config.Planet.Impact.Mars.Description,
				planet.Jupiter:   config.Config.Planet.Impact.Jupiter.Description,
				planet.Saturn:    config.Config.Planet.Impact.Saturn.Description,
				planet.Uranus:    config.Config.Planet.Impact.Uranus.Description,
				planet.Neptune:   config.Config.Planet.Impact.Neptune.Description,
				planet.Pluto:     config.Config.Planet.Impact.Pluto.Description,
				planet.Sun:       config.Config.Planet.Impact.Sun.Description,
				planet.BlackHole: config.Config.Planet.Impact.BlackHole.Description,
				planet.Supernova: config.Config.Planet.Impact.Supernova.Description,
			}[h.planet.Type]),
		},
	)

	switch h.planet.Type {
	case planet.Sun:
		// If the spaceship is within range of the sun, unfreeze the spaceship.
		if h.spaceship.State() == spaceship.Frozen && h.planet.WithinRange(h.spaceship.Geometry.Position().Add(h.spaceship.Geometry.Size().Half().ToVector()), 1) {
			h.spaceship.ResetState()
		}

		for i, e := range h.enemies {
			// If a freezer is within range of the sun, unfreeze the freezer.
			if e.Type() == enemy.Freezer && h.planet.WithinRange(h.enemies[i].Geometry.Position().Add(e.Geometry.Size().Half().ToVector()), 1) {
				h.enemies[i].ChangeType(enemy.Normal)
			}
		}

		h.planet.DoOnce(func() { config.SendMessage(message, false, false) })

	case planet.BlackHole:
		// If the spaceship is within range of the hole, disable the boost.
		if h.spaceship.State() == spaceship.Boosted && h.planet.WithinRange(h.spaceship.Geometry.Position().Add(h.spaceship.Geometry.Size().Half().ToVector()), 1) {
			h.spaceship.ResetState()
		}

		h.planet.DoOnce(func() { config.SendMessage(message, false, false) })

	case planet.Supernova:
		// Unfreeze the spaceship immediately if frozen.
		if h.spaceship.State() == spaceship.Frozen {
			h.spaceship.ResetState()
		}

		h.planet.DoOnce(func() { config.SendMessage(message, false, false) })

	case planet.Uranus, planet.Neptune:
		h.planet.DoOnce(func() {
			// Increases the specialty likeliness of the enemies for freezers.
			for i, e := range h.enemies {
				h.enemies[i].SpecialtyLikeliness = (e.SpecialtyLikeliness * numeric.Number(map[planet.PlanetType]float64{
					planet.Uranus:  config.Config.Planet.Impact.Uranus.SpecialFoeLikelinessAmplifier,
					planet.Neptune: config.Config.Planet.Impact.Neptune.SpecialFoeLikelinessAmplifier,
				}[h.planet.Type])).Clamp(0, 1)
				h.enemies[i].Surprise(enemy.Freezer, enemy.Cloaked)
			}

			config.SendMessage(message, false, false)
		})

	case planet.Mercury, planet.Mars:
		h.planet.DoOnce(func() {
			// Double the berserk likeliness of the enemies.
			for i, e := range h.enemies {
				h.enemies[i].Level.BerserkLikeliness = (e.Level.BerserkLikeliness * numeric.Number(map[planet.PlanetType]float64{
					planet.Mercury: config.Config.Planet.Impact.Mercury.BerserkLikelinessAmplifier,
					planet.Mars:    config.Config.Planet.Impact.Mars.BerserkLikelinessAmplifier,
				}[h.planet.Type])).Clamp(0, 1)
				h.enemies[i].Berserk()
			}

			config.SendMessage(message, false, false)
		})

	case planet.Pluto:
		h.planet.DoOnce(func() {
			// Increases the specialty likeliness of the enemies for freezers and going on a berserk.
			for i, e := range h.enemies {
				h.enemies[i].SpecialtyLikeliness = (e.SpecialtyLikeliness *
					numeric.Number(config.Config.Planet.Impact.Pluto.SpecialFoeLikelinessAmplifier)).Clamp(0, 1)
				h.enemies[i].Level.BerserkLikeliness = (e.Level.BerserkLikeliness *
					numeric.Number(config.Config.Planet.Impact.Pluto.BerserkLikelinessAmplifier)).Clamp(0, 1)
				h.enemies[i].Berserk()
				h.enemies[i].Surprise(enemy.Freezer, enemy.Cloaked)
			}

			config.SendMessage(message, false, false)
		})

	case planet.Jupiter, planet.Saturn:
		h.planet.DoOnce(func() {
			// Increases the defense and hitpoints of the enemies.
			for i := range h.enemies {
				h.enemies[i].Level.Defense *= map[planet.PlanetType]int{
					planet.Jupiter: config.Config.Planet.Impact.Jupiter.EnemyDefenseAmplifier,
					planet.Saturn:  config.Config.Planet.Impact.Saturn.EnemyDefenseAmplifier,
				}[h.planet.Type]
				h.enemies[i].Level.HitPoints *= map[planet.PlanetType]int{
					planet.Jupiter: config.Config.Planet.Impact.Jupiter.EnemyHitpointsAmplifier,
					planet.Saturn:  config.Config.Planet.Impact.Saturn.EnemyHitpointsAmplifier,
				}[h.planet.Type]
			}

			config.SendMessage(message, false, false)
		})

	case planet.Venus, planet.Earth:
		h.planet.DoOnce(func() {
			// Slow down the spaceship and increase the specialty likeliness for goodies.
			h.spaceship.Level.AccelerateRate *= numeric.Number(map[planet.PlanetType]float64{
				planet.Venus: config.Config.Planet.Impact.Venus.SpaceshipDeceleration,
				planet.Earth: config.Config.Planet.Impact.Earth.SpaceshipDeceleration,
			}[h.planet.Type])
			for i, e := range h.enemies {
				h.enemies[i].SpecialtyLikeliness = (e.SpecialtyLikeliness * numeric.Number(map[planet.PlanetType]float64{
					planet.Venus: config.Config.Planet.Impact.Venus.TankLikelinessAmplifier,
					planet.Earth: config.Config.Planet.Impact.Earth.TankLikelinessAmplifier,
				}[h.planet.Type])).Clamp(0, 1)
				h.enemies[i].Surprise(enemy.Tank)
			}

			config.SendMessage(message, false, false)
		})

	}
}

// checkCollisions checks if the spaceship has collided with an enemy.
// If the spaceship has collided with an enemy, it applies the necessary
// penalties and upgrades.
// If the spaceship has collided with a goodie, it upgrades the spaceship.
// If the spaceship has collided with a freezer, it freezes the spaceship.
// If the spaceship has collided with a normal enemy, it applies default penalty.
// If the spaceship has collided with a berserker, it applies the berserker penalty.
// If the spaceship has collided with an annihilator, it applies the annihilator penalty.
// If the spaceship is boosted, it destroys the enemy.
// It checks if the bullets have hit an enemy.
// If the bullets have hit an enemy, it applies the necessary damage.
// If the enemy has no health points, it upgrades the spaceship.
// If the enemy is a goodie, it does nothing.
// If the enemy is a freezer and the spaceship is not an admiral, it does nothing.
// If the enemy is a freezer and the planet is not the sun or a supernova, it does nothing.
// If the bullet has not hit the enemy, it does nothing.
func (h *handler) checkCollisions() {
	// Discover the planet.
	if h.spaceship.Discover(h.planet) {
		if discovered := h.spaceship.Discovered(); len(discovered) == planet.PlanetsCount && !h.spaceship.IsAdmiral {
			h.spaceship.IsAdmiral = true // Promote the commander to admiral
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.AllPlanetsDiscovered, config.Template{
				"PlanetName": h.planet.Type.String(),
			}), false, false)
		} else {
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.PlanetDiscovered, config.Template{
				"PlanetName":       h.planet.Type.String(),
				"RemainingPlanets": planet.PlanetsCount - len(discovered),
				"TotalPlanets":     planet.PlanetsCount,
			}), false, false)
		}
	}

	// Check if the spaceship has collided with an enemy.
	for j, e := range h.enemies {
		if e.Level.HitPoints > 0 && h.spaceship.DetectCollision(e) { // Collision detected.

			// Repel the enemy
			if config.Config.Control.RepelEnemies.Get() {
				h.enemies[j].Geometry.SetPosition(h.spaceship.ApplyRepulsion(e))
			}

			// If the spaceship is boosted, do nothing.
			if h.spaceship.State() == spaceship.Boosted && e.Type() != enemy.Tank {
				continue
			}

			h.enemies[j].Destroy() // Destroy the enemy due to the collision.
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.EnemyDestroyed, config.Template{
				"EnemyName": e.Name,
				"EnemyType": e.Type(),
			}), false, true)

			// If the spaceship is boosted, gain experience.
			if h.spaceship.State() == spaceship.Boosted {
				if e.Type() == enemy.Tank { // Prolongate the boosted state.
					h.spaceship.ChangeState(spaceship.Boosted)
				}

				if h.spaceship.Level.GainExperience(e) { // Gain experience and upgrade the spaceship.
					config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipUpgradedByEnemyKill, config.Template{
						"EnemyName":      e.Name,
						"EnemyType":      e.Type(),
						"SpaceshipLevel": h.spaceship.Level.Progress,
					}), false, true)
				}

				// Enemy has been processed, continue to the next enemy.
				continue
			}

			// Get the penalty of the enemy.
			penalty := e.Type().GetPenalty()
			// If the spaceship is frozen or hijacked, apply the penalty.
			if h.spaceship.State().AnyOf(spaceship.Frozen, spaceship.Hijacked) && penalty > 0 {
				switch {
				case // Prolongate the state.
					e.Type() == enemy.Freezer && h.spaceship.State() == spaceship.Frozen,
					e.Type() == enemy.Cloaked && h.spaceship.State() == spaceship.Hijacked:

					h.spaceship.ChangeState(h.spaceship.State())
				}

				if h.spaceship.Penalize(penalty) { // Apply the penalty.
					config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipDowngradedByEnemy, config.Template{
						"SpaceshipLevel": h.spaceship.Level.Progress,
					}), false, true)
				}

				if h.spaceship.IsDestroyed() { // Check if the spaceship has been destroyed.
					config.SendMessage(config.Execute(config.Config.MessageBox.Messages.GameOver, config.Template{
						"DiscoveredPlanets": h.spaceship.Discovered(),
						"HighScore":         h.spaceship.Level.HighScore,
						"Rank":              config.SetScore(h.spaceship.Commandant, h.spaceship.Level.HighScore),
						"TopScores":         config.GetScores(10),
					}), false, false)
					h.pause()
					h.cancel()
					return
				}

				// The enemy has been processed, continue to the next enemy.
				continue
			}

			// Change the spaceship state.
			h.spaceship.ChangeState(map[enemy.EnemyType]spaceship.SpaceshipState{
				enemy.Cloaked:     spaceship.Hijacked,
				enemy.Normal:      spaceship.Damaged,
				enemy.Freezer:     spaceship.Frozen,
				enemy.Tank:        spaceship.Boosted,
				enemy.Berserker:   spaceship.Damaged,
				enemy.Annihilator: spaceship.Damaged,
				enemy.Juggernaut:  spaceship.Damaged,
				enemy.Dreadnought: spaceship.Damaged,
				enemy.Behemoth:    spaceship.Damaged,
				enemy.Colossus:    spaceship.Damaged,
				enemy.Leviathan:   spaceship.Damaged,
				enemy.Bulwark:     spaceship.Damaged,
				enemy.Overlord:    spaceship.Damaged,
			}[e.Type()])

			// If the spaceship has been boosted, upgrade the spaceship.
			if h.spaceship.State() == spaceship.Boosted {
				config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipBoosted, config.Template{
					"SpaceshipLevel": h.spaceship.Level.Progress,
				}), false, true)

				if h.spaceship.Level.GainExperience(e) {
					config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipUpgradedByTank, config.Template{
						"EnemyName":      e.Name,
						"EnemyType":      e.Type(),
						"SpaceshipLevel": h.spaceship.Level.Progress,
					}), false, true)
				}

				// The enemy has been processed, continue to the next enemy.
				continue
			}

			if h.spaceship.State() == spaceship.Hijacked {
				config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipHijacked, config.Template{
					"EnemyName": e.Name,
					"EnemyType": e.Type(),
				}), false, true)
			}

			// Penalize the spaceship and downgrade it.
			if h.spaceship.Penalize(penalty) {
				config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipDowngradedByEnemy, config.Template{
					"SpaceshipLevel": h.spaceship.Level.Progress,
				}), false, true)
			}

			// Check if the spaceship has been destroyed.
			if h.spaceship.IsDestroyed() {
				config.SendMessage(config.Execute(config.Config.MessageBox.Messages.GameOver, config.Template{
					"DiscoveredPlanets": h.spaceship.Discovered(),
					"HighScore":         h.spaceship.Level.HighScore,
					"Rank":              config.SetScore(h.spaceship.Commandant, h.spaceship.Level.HighScore),
					"TopScores":         config.GetScores(10),
				}), false, false)
				h.pause()
				h.cancel()
				return
			}

			// Notify the user about the frozen spaceship.
			if h.spaceship.State() == spaceship.Frozen {
				config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipFrozen, config.Template{
					"SpaceshipLevel": h.spaceship.Level.Progress,
				}), false, true)
			}
		}

		// Check if the bullets have hit the enemy.
		for i, b := range h.spaceship.Bullets {
			// If the bullet has been exhausted, do nothing.
			// If the bullet has not hit the enemy, do nothing.
			if e.IsDestroyed() || !b.HasHit(e) {
				continue
			}

			// If the enemy is a goodie, repel the bullet.
			// If the enemy is a freezer and the spaceship is not an admiral and the planet is not the sun or a supernova, repel the bullet.
			if e.Type() == enemy.Tank ||
				(e.Type().AnyOf(enemy.Freezer, enemy.Cloaked) &&
					!h.spaceship.IsAdmiral && !h.planet.Type.AnyOf(planet.Sun, planet.Supernova)) {

				h.enemies[j].Geometry.SetPosition(h.spaceship.Bullets[i].Repel(e)) // Repel the bullet from the enemy.
				continue
			}

			damage := h.enemies[j].Hit(b.GetDamage()) // Apply the damage to the enemy.
			if damage == 0 {
				h.enemies[j].Geometry.SetPosition(h.spaceship.Bullets[i].Repel(e)) // Repel the bullet from the enemy.
				continue
			}

			h.spaceship.Bullets[i].Exhaust() // Exhaust the bullet.

			// Throttled: with eight cannons on an 85ms cooldown this fires about
			// ninety times a second, and every message reparses HTML and restarts
			// a smooth scroll, which drowns the log and stalls the frame.
			config.SendMessageThrottled("enemy_hit",
				config.Execute(config.Config.MessageBox.Messages.EnemyHit, config.Template{
					"EnemyName": e.Name,
					"EnemyType": e.Type(),
					"Damage":    damage,
				}), false, true, config.Config.MessageBox.ChannelLogThrottling)

			// If the enemy has no health points, upgrade the spaceship.
			if h.enemies[j].IsDestroyed() && h.spaceship.Level.GainExperience(e) {
				config.SendMessage(config.Execute(config.Config.MessageBox.Messages.SpaceshipUpgradedByEnemyKill, config.Template{
					"EnemyName":      e.Name,
					"EnemyType":      e.Type(),
					"SpaceshipLevel": h.spaceship.Level.Progress,
				}), false, true)
			}

			// If the progress is a multiple of the enemy count progress step,
			// generate a new enemy.
			if h.spaceship.Level.Progress%config.Config.Enemy.CountProgressStep == 0 &&
				len(h.enemies) < config.Config.Enemy.MaximumCount {

				h.GenerateEnemy("", false)
			}

		}
	}
}

// draw draws the game objects on the canvas.
// It clears the canvas and the background.
// It draws the stars on the background.
// It draws the background.
// It draws the spaceship.
// It draws the enemies.
// It draws the bullets.
func (h *handler) draw(scale numeric.Number) {
	config.ClearCanvas()

	// Draw stars on the background.
	// The starfield is static, so it is painted into the offscreen canvas once
	// and blotted from there by DrawBackground on every later frame.
	if h.stars.Twinkling() {
		config.ClearBackground()
		for i := range h.stars {
			h.stars[i].Draw()
			h.stars[i].Exhaust()
		}
	}

	// Draw background
	config.DrawBackground(h.spaceship.Level.AccelerateRate.Float() * config.Config.Star.SpeedRatio * scale.Float())

	// Draw planet
	h.planet.Draw()

	// Draw spaceship
	h.spaceship.Draw()

	// Draw enemies
	for _, e := range h.enemies {
		e.Draw()
	}

	// Draw bullets
	for _, b := range h.spaceship.Bullets {
		b.Draw()
	}

	for _, b := range h.enemyBullets {
		b.Draw()
	}
}

// handleKeyEvent handles the key event.
// It sets the running state to true when the key event is triggered.
// It moves the spaceship to the left when the arrow left key is pressed.
// It moves the spaceship to the right when the arrow right key is pressed.
// It moves the spaceship up when the arrow up key is pressed.
// It moves the spaceship down when the arrow down key is pressed.
// It fires bullets when the space key is pressed.
// It pauses the game when the pause key is pressed.
// It removes the key from the keysHeld map when the key is released.
func (h *handler) handleKeyEvent(key keyEvent) {
	select {
	case <-h.ctx.Done():
		return

	default:
		bound := key.Key.Action()

		if !key.Pressed {
			delete(h.keysHeld, bound)
			return
		}

		if h.start() {
			return
		}

		switch bound {
		case actionNone:

		case actionPause:
			h.pause()

		default:
			h.keysHeld[bound] = true

		}
	}
}

// handleKeyHold handles the key hold event.
// It fires bullets when the space key is held.
func (h *handler) handleKeyhold(scale numeric.Number) {
	select {
	case <-h.ctx.Done():
		return

	default:
		for _, held := range heldActions {
			if !h.keysHeld[held] {
				continue
			}

			switch held {
			case actionMoveDown:
				h.spaceship.MoveDown(scale)

			case actionMoveLeft:
				h.spaceship.MoveLeft(scale)

			case actionMoveRight:
				h.spaceship.MoveRight(scale)

			case actionMoveUp:
				h.spaceship.MoveUp(scale)

			case actionFire:
				h.spaceship.Fire()

			}
		}
	}
}

// handleMouse handles the mouse event.
// It sets the running state to true when the mouse event is triggered.
// It moves the spaceship to the left when the delta X is negative.
// It moves the spaceship to the right when the delta X is positive.
// It pauses the game when the auxiliary or secondary button is pressed.
func (h *handler) handleMouse(event mouseEvent) {
	select {
	case <-h.ctx.Done():
		return

	default:
		if !event.Pressed { // If the mouse button is released, do nothing.
			delete(h.mouseHeld, event.Button)
			return
		}

		if h.start() { // If the game has just started, do nothing.
			return
		}

		switch event.Button {
		case MouseButtonPrimary: // pass through

		case MouseButtonAuxiliary, MouseButtonSecondary: // If the auxiliary or secondary button is pressed, pause the game.
			h.pause()
			return

		default: // Do nothing for any other button.
			return

		}

		switch event.Type {
		case MouseEventTypeDown:
			h.mouseHeld[event.Button] = true
			return

		case MouseEventTypeUp:
			delete(h.mouseHeld, event.Button)
			return

		}

		// handling of mouse move event
		h.mouseHeld[event.Button] = true // make sure the button is held (if button down event has been missed)
		h.handleMoveEventTypes(event.CurrentPosition, event.StartPosition, 1)
	}
}

// handleMouseHeld handles the mouse held event.
// It fires bullets when the primary mouse button is held.
func (h *handler) handleMouseHeld() {
	select {
	case <-h.ctx.Done():
		return

	default:
		if h.mouseHeld[MouseButtonPrimary] {
			h.spaceship.Fire()
		}
	}
}

// handleTouch handles the touch event.
// It sets the running state to true when the touch event is triggered.
// It moves the spaceship to the left when the delta X is negative.
// It moves the spaceship to the right when the delta X is positive.
// It pauses the game when the multi-tap event is triggered.
func (h *handler) handleTouch(event touchEvent) {
	select {
	case <-h.ctx.Done():
		return

	default:
		if h.start() { // If the game has just started, do nothing.
			return
		}

		// If there are correlated touch events, pause the game.
		if event.MultiTap {
			h.pause()
			return
		}

		switch event.Type {
		case TouchTypeStart:
			h.touchHeld = true
			return

		case TouchTypeEnd:
			h.touchHeld = false
			return

		}

		// handle touch move event
		h.touchHeld = true // make sure the touch is held (if touch down event has been missed)
		h.handleMoveEventTypes(event.CurrentPosition, event.StartPosition, 1)
	}
}

// handleMoveEventTypes handles the move event types (mouse move event and touch move event).
// It moves the spaceship to the position.
// If the current position is not zero, it moves the spaceship to the current position.
// If the start position is not zero, it moves the spaceship to the start position.
// It corrects the position by the canvas dimensions.
func (h *handler) handleMoveEventTypes(eventCurrentPosition, eventStartPosition numeric.Position, scale numeric.Number) {
	canvasDimensions := config.CanvasBoundingBox()
	positionCorrection := numeric.Locate(canvasDimensions.ScaleWidth, canvasDimensions.ScaleHeight)

	switch {
	case !eventCurrentPosition.IsZero():
		h.spaceship.MoveTo(eventCurrentPosition.DivX(positionCorrection), scale)

	case !eventStartPosition.IsZero():
		h.spaceship.MoveTo(eventStartPosition.DivX(positionCorrection), scale)

	}
}

// handleTouchHeld handles the touch held event.
// It fires bullets when the touch is held.
func (h *handler) handleTouchHeld() {
	select {
	case <-h.ctx.Done():
		return

	default:
		if !h.touchHeld {
			return
		}

		h.spaceship.Fire()
	}
}

// pause pauses the game.
func (h *handler) pause() {
	if !running.Get(h.ctx) { // If the game is not running, do nothing.
		return
	}

	paused.Set(&h.ctx, true)     // signal that the game is paused
	running.Set(&h.ctx, false)   // signal that the game is not running
	suspended.Set(&h.ctx, false) // signal that the game is not suspended

	config.SendMessage(config.Execute(config.Config.MessageBox.Messages.GamePaused), false, false)
}

// render is a method that renders the game.
// It draws the spaceship, bullets and enemies on the canvas.
// The spaceship is drawn in white color.
// The bullets are drawn in yellow color.
// The enemies are drawn in gray color.
// The goodie enemies are drawn in green color.
// The berserker enemies are drawn in red color.
// The annihilator enemies are drawn in dark red color.
// The spaceship is drawn in dark red color if it is damaged.
// The spaceship is drawn in yellow color if it is boosted.
// The spaceship is drawn in white color if it is normal.
// If draws objects as rectangles.
func (h *handler) render(scale numeric.Number) {
	switch {
	case
		offline.Get(h.ctx),                             // If the game is offline, do nothing.
		suspended.Get(h.ctx),                           // If the game is suspended, do nothing.
		!running.Get(h.ctx) && !isFirstTime.Get(h.ctx): // If the game is not running and not the first time, do nothing.

		return
	}

	h.draw(scale)

	config.UpdateHUD(config.HUD{
		Score:          h.spaceship.Level.HighScore,
		Level:          h.spaceship.Level.Progress,
		Cannons:        h.spaceship.Level.Cannons,
		ShieldCharge:   h.spaceship.Level.Shield.Charge,
		ShieldCapacity: h.spaceship.Level.Shield.Capacity,
		Experience:     h.spaceship.Level.ExperienceRatio(),
	})
}

// refresh refreshes the game state.
// It updates the bullets of the spaceship.
// It updates the enemies.
// It updates the state of the spaceship.
// It checks the collisions.
func (h *handler) refresh(scale numeric.Number) {
	switch {
	case
		offline.Get(h.ctx),   // If the game is offline, do nothing.
		suspended.Get(h.ctx), // If the game is suspended, do nothing.
		!running.Get(h.ctx):  // If the game is not running, do nothing.

		return
	}

	// Update the positions of the enemies.
	h.enemies.Update(h.spaceship.Geometry.Position(), scale)

	// Update the position of the planet.
	h.planet.Update(h.spaceship.Level.AccelerateRate * numeric.Number(config.Config.Planet.SpeedRatio) * scale)

	// Update the state of the spaceship.
	h.spaceship.UpdateState(scale)

	// Recharge the shield of the spaceship.
	h.spaceship.Level.Shield.Recharge()

	// Update the positions of the bullets.
	h.spaceship.Bullets.Update(scale)
	h.enemyBullets.Update(scale)

	// Let the armed enemies return fire.
	h.fireEnemyCannons()

	// Apply the impact of the planet on the system.
	h.applyPlanetImpact()

	// Check the collisions.
	h.checkCollisions()
	h.checkEnemyFire()
}

// fireEnemyCannons lets every armed enemy take its shot.
// The bullets are held by the handler rather than by the enemies: the bullet
// package resolves collisions against enemies, so an enemy owning its own bullets
// would close an import cycle.
func (h *handler) fireEnemyCannons() {
	for i := range h.enemies {
		origin, damage, fired := h.enemies[i].FireCannon()
		if !fired {
			continue
		}

		h.enemyBullets.ReloadHostile(origin, damage, 0, h.enemies[i].Level.Speed)
	}
}

// checkEnemyFire applies the damage of the enemy bullets that reached the spaceship.
func (h *handler) checkEnemyFire() {
	for i := range h.enemyBullets {
		if h.enemyBullets[i].Exhausted || !h.enemyBullets[i].HasHitSpaceship(h.spaceship.Geometry.Position(), h.spaceship.Geometry.Size()) {
			continue
		}

		h.enemyBullets[i].Exhaust()

		// A shot costs the spaceship a level, exactly as a collision does, so the
		// shield stays the thing that absorbs it.
		if !h.spaceship.Penalize(1) {
			continue
		}

		config.SendMessageThrottled("spaceship_shot",
			config.Execute(config.Config.MessageBox.Messages.SpaceshipDowngradedByEnemy, config.Template{
				"SpaceshipLevel": h.spaceship.Level.Progress,
			}), false, true, config.Config.MessageBox.ChannelLogThrottling)

		if h.spaceship.IsDestroyed() {
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.GameOver, config.Template{
				"DiscoveredPlanets": h.spaceship.Discovered(),
				"HighScore":         h.spaceship.Level.HighScore,
				"Rank":              config.SetScore(h.spaceship.Commandant, h.spaceship.Level.HighScore),
				"TopScores":         config.GetScores(10),
			}), false, false)
			h.pause()
			h.cancel()
			return
		}
	}
}

// start starts the game if not already started.
func (h *handler) start() bool {
	switch {
	case
		offline.Get(h.ctx),   // If the game is offline, do nothing.
		suspended.Get(h.ctx): // If the game is suspended, do nothing.

		return true
	}

	if !running.Get(h.ctx) { // If the game is not running, start the game.
		running.Set(&h.ctx, true) // signal that the game is running
		paused.Set(&h.ctx, false) // signal that the game is not paused

		if isFirstTime.Get(h.ctx) {
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.GameStarted), false, false)
		}

		go config.PlayAudio("theme_heroic.wav", true)

		return true
	}

	return false
}

// Await waits for the handler to finish and executes the shutdown function.
func (h *handler) Await() {
	<-h.ctx.Done()
	go config.StopAudio("theme_heroic.wav")
}

// GenerateEnemy generates a new enemy with the specified name and random Y position.
func (h *handler) GenerateEnemy(name string, randomY bool) { h.enemies.AppendNew(name, randomY) }

// GenerateEnemies generates the specified number of enemies with random Y position.
func (h *handler) GenerateEnemies(num int, randomY bool) {
	for i := 0; i < num; i++ {
		h.enemies.AppendNew("", randomY)
	}
}

// Loop starts the game loop.
// It refreshes the game state, renders the game, and handles the keydown events.
// It should be called in a separate goroutine.
func (h *handler) Loop() {
	frame := frames()

	if isFirstTime.Get(h.ctx) {
		config.SendMessage(config.Execute(config.Config.MessageBox.Messages.Greeting, config.Template{
			"Commandant": h.spaceship.Commandant,
		}), true, false)
	}

	// Notify the user about how to start the game.
	if !running.Get(h.ctx) {
		if isFirstTime.Get(h.ctx) {
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.ExplainInterface), false, false)
		} else {
			config.SendMessage(config.Execute(config.Config.MessageBox.Messages.HowToRestart), false, false)
		}
	}

	// Wait for the initial user input.
	for !running.Get(h.ctx) {
		select {
		case <-h.ctx.Done():
			return

		case scale := <-frame:
			h.render(scale)

		case key := <-h.keyEvent:
			h.handleKeyEvent(key)

		case event := <-h.mouseEvent:
			h.handleMouse(event)

		case event := <-h.touchEvent:
			h.handleTouch(event)

		}
	}

	for {
		select {
		case <-h.ctx.Done():
			return

		case scale := <-frame:
			h.refresh(scale)
			h.render(scale)
			h.handleKeyhold(scale)
			h.handleMouseHeld()
			h.handleTouchHeld()

		case key := <-h.keyEvent:
			h.handleKeyEvent(key)

		case event := <-h.mouseEvent:
			h.handleMouse(event)

		case event := <-h.touchEvent:
			h.handleTouch(event)

		}
	}
}

// Restart restarts the game.
func (h *handler) Restart() {
	h.spaceship = spaceship.Embark(h.spaceship.Commandant)
	h.enemies = nil
	h.enemyBullets = nil
	h.stars = star.Explode(config.Config.Star.Count)
	h.planet = planet.Reveal(true, true)
	h.ctx, h.cancel = context.WithCancel(context.Background())
	running.Set(&h.ctx, false)
	isFirstTime.Set(&h.ctx, false)
}

// New creates a new handler.
// It creates a new spaceship and registers all event handlers.
func New() *handler {
	h := &handler{
		keyEvent:   make(chan keyEvent),
		keysHeld:   make(map[action]bool),
		mouseEvent: make(chan mouseEvent),
		mouseHeld:  make(map[mouseButton]bool),
		touchEvent: make(chan touchEvent),
		touchHeld:  false,
		planet:     planet.Reveal(true, true),
		spaceship:  spaceship.Embark(""),
		stars:      star.Explode(config.Config.Star.Count),
	}

	h.ctx, h.cancel = context.WithCancel(context.Background())
	running.Set(&h.ctx, false)
	isFirstTime.Set(&h.ctx, true)
	h.registerEventHandlers()
	h.ask()

	return h
}
