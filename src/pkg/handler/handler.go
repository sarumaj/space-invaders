package handler

import (
	"context"
	"sync"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/bullet"
	"github.com/sarumaj/edu-space-invaders/src/pkg/objects/effect"
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
	blasts       effect.Blasts        // blasts are the destruction animations currently playing
	keyEvent     chan keyEvent        // keyupEvent is the channel for key events
	keysHeld     map[action]bool      // keysHeld is the set of actions whose key is currently down
	mouseEvent   chan mouseEvent      // mouseEvent is the channel for mouse events
	mouseHeld    map[mouseButton]bool // mouseHeld is the map of mouse buttons held
	once         sync.Once            // once is meant to register the keydown event only once
	planet       *planet.Planet       // planet is the planet to be drawn
	pointer      *numeric.Position    // pointer is where the mouse or the finger is steering the spaceship, if either is down
	releaseEvent chan struct{}        // releaseEvent asks the loop to drop every held input, when the page loses focus
	spaceship    *spaceship.Spaceship // spaceship is the player's spaceship
	stars        star.Stars           // stars is the list of stars
	touchEvent   chan touchEvent      // touchEvent is the channel for touch events
	touchHeld    bool                 // touchHeld is the flag to indicate if the touch is held
	wreckage     *wreckage            // wreckage is the spaceship's destruction sequence, set only once the game has been lost
}

// wreckage is the spaceship's destruction sequence. Losing used to stop the game
// on the same frame the last level was taken, so the spaceship simply vanished
// under the mission report; this keeps the game drawing for as long as the wreck
// is still coming apart, and holds the report back until it has.
type wreckage struct {
	origin    numeric.Position // Centre of the hull that came apart.
	radius    numeric.Number   // Radius of that hull.
	color     string           // Colour the spaceship was wearing when it died.
	epitaph   string           // Mission report, composed at the moment of death and shown once the fire goes out.
	age, life numeric.Number   // How many nominal frames the sequence has run, and how many it runs for.
	spawned   int              // Secondary explosions started so far.
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
			h.destroy(config.Execute(config.Config.Planet.Impact.BlackHole.SpaceshipDestroyedReason))
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
					h.destroy("")
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
				h.destroy("")
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
			// generate a new enemy. The check lives in the bullet loop, so it holds
			// for every hit landed until the next level: the fleet is sized against
			// the progress reached rather than the number of times it was noticed.
			if wanted := config.Config.Enemy.Count + h.spaceship.Level.Progress/config.Config.Enemy.CountProgressStep; len(h.enemies) < wanted &&
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

	// Draw spaceship, unless it is the wreck that is being drawn instead.
	if !h.dying() {
		h.spaceship.Draw(scale)
	}

	// Draw enemies
	for i := range h.enemies {
		h.enemies[i].Draw(scale)
	}

	// Draw bullets
	for _, b := range h.spaceship.Bullets {
		b.Draw()
	}

	for _, b := range h.enemyBullets {
		b.Draw()
	}

	// Draw the destruction animations last, so they read as being in front.
	h.blasts.Draw()
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
		if h.dying() { // Nothing the player does can steer a wreck.
			return
		}

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

// handleHeld applies everything that is currently held down, once per frame.
// The three input methods share this one place on purpose. Holding a movement
// key was applied here, per frame and scaled by the frame time, while dragging
// the mouse or a finger was applied straight from the listener, once per input
// event and unscaled — so the spaceship covered ground at the rate the device
// happened to report at, and a 1000 Hz mouse outran the keyboard by more than an
// order of magnitude. Firing was split the same way. Everything held now goes
// through one frame-scaled path, which is what makes the three feel alike.
func (h *handler) handleHeld(scale numeric.Number) {
	select {
	case <-h.ctx.Done():
		return

	default:
	}

	switch {
	case
		h.dying(),           // A wreck takes no orders.
		!running.Get(h.ctx): // Held input must not fly or fire the spaceship while the game is paused.

		return
	}

	// Steer towards the pointer, if the mouse button or a finger is down.
	if h.pointer != nil {
		h.spaceship.MoveTo(*h.pointer, scale)
	}

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

	// Holding the primary mouse button or a finger fires exactly as holding the
	// fire key does; the cooldown in Fire is what limits all three.
	if h.mouseHeld[MouseButtonPrimary] || h.touchHeld {
		h.spaceship.Fire()
	}
}

// handleMouse handles the mouse event.
// It sets the running state to true when the mouse event is triggered.
// It steers the spaceship towards the pointer while the primary button is down.
// It pauses the game when the auxiliary or secondary button is pressed.
func (h *handler) handleMouse(event mouseEvent) {
	select {
	case <-h.ctx.Done():
		return

	default:
		if h.dying() { // Nothing the player does can steer a wreck.
			return
		}

		if !event.Pressed { // If the mouse button is released, do nothing.
			delete(h.mouseHeld, event.Button)
			h.pointer = nil
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
			// Aim at the press itself, so that pressing and holding without moving
			// steers, exactly as holding a finger down does.
			h.mouseHeld[event.Button] = true
			h.aimAt(event.StartPosition, event.StartPosition)
			return

		case MouseEventTypeUp:
			delete(h.mouseHeld, event.Button)
			h.pointer = nil
			return

		}

		// handling of mouse move event
		h.mouseHeld[event.Button] = true // make sure the button is held (if button down event has been missed)
		h.aimAt(event.CurrentPosition, event.StartPosition)
	}
}

// handleTouch handles the touch event.
// It sets the running state to true when the touch event is triggered.
// It steers the spaceship towards the finger while it is on the screen.
// It pauses the game when two fingers are put down at once.
func (h *handler) handleTouch(event touchEvent) {
	select {
	case <-h.ctx.Done():
		return

	default:
		if h.dying() { // Nothing the player does can steer a wreck.
			return
		}

		if h.start() { // If the game has just started, do nothing.
			return
		}

		switch event.Type {
		case TouchTypeStart:
			// Only a deliberate two-finger tap pauses. Testing MultiTap on every
			// touch event meant a second finger brushing the screen mid-drag
			// paused the game, and left the first finger registered as held.
			if event.MultiTap {
				h.pause()
				return
			}

			h.touchHeld = true
			h.aimAt(event.StartPosition, event.StartPosition)
			return

		case TouchTypeEnd:
			h.touchHeld = false
			h.pointer = nil
			return

		}

		// handle touch move event
		h.touchHeld = true // make sure the touch is held (if touch down event has been missed)
		h.aimAt(event.CurrentPosition, event.StartPosition)
	}
}

// aimAt records where the pointer wants the spaceship to be, correcting for the
// canvas scale. The steering itself waits for the next frame, in handleHeld, so
// that it is applied once per frame however often the device reports.
func (h *handler) aimAt(eventCurrentPosition, eventStartPosition numeric.Position) {
	canvasDimensions := config.CanvasBoundingBox()
	positionCorrection := numeric.Locate(canvasDimensions.ScaleWidth, canvasDimensions.ScaleHeight)

	var target numeric.Position
	switch {
	case !eventCurrentPosition.IsZero():
		target = eventCurrentPosition.DivX(positionCorrection)

	case !eventStartPosition.IsZero():
		target = eventStartPosition.DivX(positionCorrection)

	default:
		return
	}

	h.pointer = &target
}

// releaseAll drops every held input. The browser stops delivering key and
// pointer events as soon as the page loses focus or the system takes a gesture
// over, and never sends the matching release, so without this the spaceship
// flies and fires on input the player let go of long ago.
func (h *handler) releaseAll() {
	clear(h.keysHeld)
	clear(h.mouseHeld)
	h.touchHeld = false
	h.pointer = nil
}

// pause pauses the game.
func (h *handler) pause() {
	if !running.Get(h.ctx) { // If the game is not running, do nothing.
		return
	}

	paused.Set(&h.ctx, true)     // signal that the game is paused
	running.Set(&h.ctx, false)   // signal that the game is not running
	suspended.Set(&h.ctx, false) // signal that the game is not suspended

	// The gesture that paused is still physically held, and the one that resumes
	// is consumed by start, so anything left held here would be applied to a
	// spaceship the player is not looking at.
	h.releaseAll()

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

	// Start a destruction animation for whatever died this frame, and age the
	// ones already running.
	h.collectBlasts()
	h.blasts.Update(scale)
}

// dying reports whether the spaceship's destruction sequence is playing out.
func (h *handler) dying() bool { return h.wreckage != nil }

// destroy ends the game. The mission report is composed here, so that the score
// is recorded at the moment of death, but it is held back until the spaceship has
// finished coming apart: showing it on the same frame the last level was taken
// left the player reading a scoreboard over a spaceship that had simply vanished.
// The reason, when there is one, says what killed the spaceship.
func (h *handler) destroy(reason string) {
	if h.dying() { // The spaceship only dies once.
		return
	}

	h.releaseAll()

	report := config.Template{
		"DiscoveredPlanets": h.spaceship.Discovered(),
		"HighScore":         h.spaceship.Level.HighScore,
		"Rank":              config.SetScore(h.spaceship.Commandant, h.spaceship.Level.HighScore),
		"TopScores":         config.GetScores(10),
	}

	// The template falls back to a generic epitaph on a missing reason, and an
	// empty string is not missing enough for it.
	if reason != "" {
		report["Reason"] = reason
	}

	size := h.spaceship.Geometry.Size()
	h.wreckage = &wreckage{
		origin:  h.spaceship.Geometry.Position().Add(size.Half().ToVector()),
		radius:  size.ToVector().Magnitude() / 2,
		color:   h.spaceship.Color.Gradient().FormatRGBA(),
		epitaph: config.Execute(config.Config.MessageBox.Messages.GameOver, report),
		life: numeric.Number(config.Config.Effect.WreckDuration.Seconds() *
			config.Config.Control.DesiredFramesPerSecondRate),
	}

	// The hull goes first and large, over the whole sequence; the secondary
	// explosions follow in mourn.
	h.blasts.DetonateFor(
		h.wreckage.origin,
		h.wreckage.radius*numeric.Number(config.Config.Effect.WreckScale),
		config.BlastWreck,
		h.wreckage.color,
		config.Config.Effect.WreckDuration,
	)

	go config.PlayAudio("spaceship_crash.wav", false)
	go config.StopAudio("theme_heroic.wav")
}

// mourn advances the destruction sequence by one frame and closes the game once
// the wreck has burnt out. The world is held still while it plays: an enemy
// gliding on through the explosion reads as the game having carried on without
// the player.
func (h *handler) mourn(scale numeric.Number) {
	h.wreckage.age += scale

	// Stagger the secondary explosions across the opening of the sequence, so the
	// hull reads as coming apart piece by piece rather than in a single flash.
	// They are all started early enough to burn out before the report arrives:
	// cutting one off mid-animation is what would make the report feel abrupt.
	const spawnWindow = 0.6 // Fraction of the sequence the secondaries are started within.
	for count := config.Config.Effect.WreckBlasts; h.wreckage.spawned < count &&
		h.wreckage.age*numeric.Number(count) >= h.wreckage.life*spawnWindow*numeric.Number(h.wreckage.spawned); h.wreckage.spawned++ {

		h.blasts.Detonate(
			h.wreckage.origin.Add(numeric.Locate(
				numeric.RandomRange(-h.wreckage.radius, h.wreckage.radius),
				numeric.RandomRange(-h.wreckage.radius, h.wreckage.radius),
			)),
			h.wreckage.radius*numeric.RandomRange(0.35, 0.8),
			[...]string{config.BlastBurst, config.BlastShatter, config.BlastShockwave}[h.wreckage.spawned%3],
			h.wreckage.color,
		)
	}

	h.blasts.Update(scale)
	h.draw(scale)

	if h.wreckage.age < h.wreckage.life {
		return
	}

	config.SendMessage(h.wreckage.epitaph, false, false)

	// The state changes of a pause, without its message: the mission report has
	// just gone out, and an invitation to take a break underneath it reads as
	// noise. The loop that restarts the game explains how to play again.
	paused.Set(&h.ctx, true)
	running.Set(&h.ctx, false)
	suspended.Set(&h.ctx, false)
	h.cancel()
}

// collectBlasts starts a destruction animation for every enemy that died since
// the last frame. Enemies are destroyed by collisions, by bullets and by the
// black hole, so they are collected here rather than at each of those sites.
func (h *handler) collectBlasts() {
	for i := range h.enemies {
		if !h.enemies[i].Detonate() {
			continue
		}

		h.blasts.Detonate(
			h.enemies[i].Geometry.Position().Add(h.enemies[i].Geometry.Size().Half().ToVector()),
			h.enemies[i].Geometry.Size().ToVector().Magnitude()/2,
			h.enemies[i].Type().GetBlast(),
			h.enemies[i].Type().GetColor().FormatRGBA(),
		)
	}
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
			h.destroy("")
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

		case <-h.releaseEvent:
			h.releaseAll()

		}
	}

	for {
		select {
		case <-h.ctx.Done():
			return

		case scale := <-frame:
			// Once the spaceship is lost the frame belongs to the wreck alone: the
			// world stops advancing, and the loop ends when the fire goes out.
			if h.dying() {
				h.mourn(scale)
				continue
			}

			h.refresh(scale)
			h.render(scale)
			h.handleHeld(scale)

		case key := <-h.keyEvent:
			h.handleKeyEvent(key)

		case event := <-h.mouseEvent:
			h.handleMouse(event)

		case event := <-h.touchEvent:
			h.handleTouch(event)

		case <-h.releaseEvent:
			h.releaseAll()

		}
	}
}

// Restart restarts the game.
func (h *handler) Restart() {
	h.spaceship = spaceship.Embark(h.spaceship.Commandant)
	h.enemies = nil
	h.enemyBullets = nil
	h.blasts = nil
	h.wreckage = nil
	h.releaseAll()
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
		// Buffered, so that the listener can drop a release in without waiting for
		// the loop: it runs on the JavaScript event loop, which the loop's own
		// frames come from.
		releaseEvent: make(chan struct{}, 1),
		planet:       planet.Reveal(true, true),
		spaceship:    spaceship.Embark(""),
		stars:        star.Explode(config.Config.Star.Count),
	}

	h.ctx, h.cancel = context.WithCancel(context.Background())
	running.Set(&h.ctx, false)
	isFirstTime.Set(&h.ctx, true)
	h.registerEventHandlers()
	h.ask()

	return h
}
