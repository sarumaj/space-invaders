package graphics

import (
	"time"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// ColorTransition represents a color transition.
type ColorTransition struct {
	animationDuration time.Duration          // Animation duration of the transition
	currentColor      Color                  // Current color used for the beginning of the transition
	targetColor       Color                  // Target color used for the end of the transition
	currentGradient   Color                  // Intermediate color resulting from the transition
	transitionEnd     func(*ColorTransition) // Transition end callback
	immutable         bool                   // If immutable, the transition cannot be changed, until it ends
}

// Gradient returns the current gradient color of the transition.
func (t *ColorTransition) Gradient() Color {
	return t.currentGradient
}

// Interpolate advances the transition by one frame.
// The scale is how far the frame advanced the simulation, expressed in nominal
// frames, so that the transition takes its configured duration in wall-clock
// time whatever the display refresh rate happens to be. Stepping by a fixed
// amount per call, as this did, ran every colour animation 2.4 times too fast on
// a 144 Hz display.
func (t *ColorTransition) Interpolate(scale numeric.Number) {
	if t.currentGradient.Equal(t.targetColor) {
		// The callback fires once and is then dropped. Left in place it was
		// re-entered on every later frame, and a callback that sets a colour would
		// keep overwriting whatever was set after it.
		if end := t.transitionEnd; end != nil {
			t.transitionEnd = nil
			end(t)
		}

		return
	}

	numberOfFrames := numeric.Number(config.Config.Control.DesiredFramesPerSecondRate) *
		numeric.Number(t.animationDuration.Seconds())

	if numberOfFrames <= 0 { // Nothing to animate over: arrive at once.
		t.currentGradient = t.targetColor
		return
	}

	for i := range t.currentGradient {
		step := (t.targetColor[i] - t.currentColor[i]) / numberOfFrames * scale

		// Land on the target rather than past it: a fractional frame scale would
		// otherwise overshoot, and the equality above would never hold again.
		if remaining := t.targetColor[i] - t.currentGradient[i]; step.Abs() >= remaining.Abs() {
			t.currentGradient[i] = t.targetColor[i]
			continue
		}

		t.currentGradient[i] += step
	}
}

// SetAnimationDuration sets the animation duration of the transition.
func (t *ColorTransition) SetAnimationDuration(duration time.Duration) *ColorTransition {
	t.animationDuration = duration
	return t
}

// SetColor sets the target color of the transition.
// Current gradient is used as the starting point of the transition.
func (t *ColorTransition) SetColor(to Color) *ColorTransition {
	if t.immutable {
		return t
	}

	if !t.targetColor.Equal(to) {
		t.currentColor, t.targetColor = t.currentGradient, to
	}

	return t
}

// SetGradient allows to set the current gradient color of the transition.
func (t *ColorTransition) SetGradient(color Color) *ColorTransition {
	t.currentGradient = color
	return t
}

// SetImmutable sets the immutable property of the transition.
func (t *ColorTransition) SetImmutable(immutable bool) *ColorTransition {
	t.immutable = immutable
	return t
}

// SetTransitionEnd sets the transition end callback.
func (t *ColorTransition) SetTransitionEnd(transitionEnd func(*ColorTransition)) *ColorTransition {
	t.transitionEnd = transitionEnd
	return t
}

// InitialColorTransition initializes a color transition with the given color.
func InitialColorTransition(color Color) *ColorTransition {
	return &ColorTransition{
		animationDuration: config.Config.Control.AnimationDuration,
		currentColor:      color,
		targetColor:       color,
		currentGradient:   color,
	}
}
