package planet

import (
	"fmt"
	"sync"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// PlanetType represents the type of a planet or other celestial body.
type Planet struct {
	Position       numeric.Position
	Radius         numeric.Number
	Type           PlanetType
	additionalMass numeric.Number
	once           *sync.Once
	spin           numeric.Number // Angle the planet is currently turned by, in radians.
	sprite         config.Sprite  // Pre-rendered image of the planet, empty for the animated types.
}

// Again reveals the planet yet again as new planet.
// The planet will be revealed at the top of the canvas.
func (planet *Planet) Again() {
	newPlanet := Reveal(false, false)
	planet.Position = newPlanet.Position
	planet.Radius = newPlanet.Radius
	planet.Type = newPlanet.Type
	planet.additionalMass = 0
	planet.once = &sync.Once{}

	// The new planet has its own type and radius, so the cached image of the old
	// one no longer describes it.
	planet.spin, planet.sprite = 0, config.Sprite{}
}

// ApplyGravity applies the gravitational force of the planet to the point.
// The center is the center of the object that the planet is attracting.
// The mass is the mass of the object that the planet is attracting.
// The additionalMass flag indicates whether the mass should contribute persistently to the planet's gravitational force.
// The function returns the new position of the point after applying the gravitational force.
func (planet *Planet) ApplyGravity(center numeric.Position, mass numeric.Number, additionalMass, reverse bool) numeric.Position {
	// Compute the distance between the planet and the point
	delta := planet.Position.Sub(center)
	distance := delta.Magnitude()

	// If the distance is zero, the point is already on the planet
	if numeric.Equal(distance, 0, 1e-3) {
		if additionalMass {
			planet.additionalMass += mass
		}

		return center
	}

	// The product of the mass and planet's mass by using the area of the planet
	massProduct := numeric.Pi * planet.Radius.Pow(2) * (mass + planet.additionalMass)

	// The gravitational field strength
	fieldStrength := massProduct / distance.Pow(2)
	if factor, ok := map[PlanetType]float64{
		BlackHole: config.Config.Planet.Impact.BlackHole.GravityStrength,
		Supernova: config.Config.Planet.Impact.Supernova.GravityStrength,
		Sun:       config.Config.Planet.Impact.Sun.GravityStrength,
	}[planet.Type]; ok {
		fieldStrength *= numeric.Number(factor)
	} else {
		fieldStrength *= numeric.Number(config.Config.Planet.Impact.DefaultGravityStrength)
	}

	if reverse {
		fieldStrength *= -1
	}

	// Ensure that fieldStrength does not exceed the distance
	if fieldStrength > distance {
		fieldStrength = distance // Clamp the movement to exactly reach the planet's position
	}

	// Update the position
	return center.Add(delta.Normalize().Mul(fieldStrength))
}

// Area returns the area of the planet.
func (planet Planet) Area() numeric.Number { return numeric.Pi * planet.Radius.Pow(2) }

// DoOnce executes the action only once during the lifetime of the planet.
func (planet *Planet) DoOnce(action func()) { planet.once.Do(action) }

// Draw draws the planet on the canvas.
// The planet will be drawn at the specified position with the specified radius.
// The planet will be drawn with the specified type.
func (planet *Planet) Draw() {
	// The sun and the anomalies change from frame to frame, so they are painted
	// directly; everything else is painted once and then blitted, which is what
	// makes turning it affordable.
	if planet.Type.Animated() {
		planet.Type.Draw(planet.Position, planet.Radius)
		return
	}

	if !planet.sprite.Valid() {
		planet.sprite = planet.Type.Sprite(planet.Radius)
	}

	config.DrawSprite(planet.sprite, planet.Position.Pack(), planet.spin.Float())
}

// String returns the string representation of the planet.
func (planet Planet) String() string {
	return fmt.Sprintf("Planet{%s, R:%f, %s}", planet.Position, planet.Radius, planet.Type)
}

// Update updates the planet position.
// The speed parameter is the speed at which the planet moves.
// If the planet reaches the bottom of the canvas, it will be reborn.
func (planet *Planet) Update(speed numeric.Number) {
	planet.Position.Y += speed
	planet.spin += numeric.Number(config.Config.Planet.SpinRate) * speed

	canvasDimensions := config.CanvasBoundingBox()
	if (planet.Position.Y - planet.Radius).Float() > canvasDimensions.OriginalHeight {
		planet.Again()
	}
}

// WithinRange returns true if the position is within the planet's range.
func (planet Planet) WithinRange(center numeric.Position, factor numeric.Number) bool {
	return planet.Position.Sub(center).Magnitude() < planet.Radius*factor
}

// Reveal reveals a new planet.
// If randomY is true, the planet will be revealed at a random Y position.
// Otherwise, the planet will be revealed at the top of the canvas.
// The planet will have a random radius and type.
func Reveal(randomY, planetsOnly bool) *Planet {
	canvasDimensions := config.CanvasBoundingBox()
	planet := &Planet{
		Position: numeric.Locate(numeric.RandomRange(0, canvasDimensions.OriginalWidth), 0),
		Radius:   numeric.RandomRange(config.Config.Planet.MinimumRadius, config.Config.Planet.MaximumRadius),
		Type:     PlanetType(numeric.RandomRange(Mercury, Supernova).Int()),
		once:     &sync.Once{},
	}

	if planetsOnly && !planet.Type.IsPlanet() {
		planet.Type = PlanetType(numeric.RandomRange(Mercury, Neptune).Int())
	}

	planet.Position.Y = -planet.Radius
	if randomY {
		planet.Position.Y = numeric.RandomRange(0, canvasDimensions.OriginalHeight)
	}

	choice := PlanetType(config.Config.Control.PlanetChoice.Get())
	if choice >= Mercury && choice <= Supernova {
		planet.Type = PlanetType(choice)
	}

	return planet
}
