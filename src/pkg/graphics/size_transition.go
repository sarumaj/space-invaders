package graphics

import (
	"time"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// Transition represents a color and object size transition.
type SizeTransition struct {
	animationDuration time.Duration         // Animation duration of the transition
	currentScale      numeric.Number        // Current scale used for the beginning of the transition
	targetScale       numeric.Number        // Target scale used for the end of the transition
	size              numeric.Size          // Current size resulting from the transition
	position          numeric.Position      // Current position resulting from the transition
	transitionEnd     func(*SizeTransition) // Transition end callback
	immutable         bool                  // If immutable, the transition cannot be changed, until it ends
}

// Interpolate advances the transition by one frame.
// The scale is how far the frame advanced the simulation, expressed in nominal
// frames, so that the transition takes its configured duration in wall-clock
// time whatever the display refresh rate happens to be.
func (t *SizeTransition) Interpolate(scale numeric.Number) {
	if numeric.Equal(t.currentScale, t.targetScale, 1e-9) {
		// The callback fires once and is then dropped, as for a colour transition.
		if end := t.transitionEnd; end != nil {
			t.transitionEnd = nil
			end(t)
		}

		return
	}

	numberOfFrames := numeric.Number(config.Config.Control.DesiredFramesPerSecondRate) *
		numeric.Number(t.animationDuration.Seconds())

	sizeFactor := t.targetScale / t.currentScale // Arrive at once, unless the steps below say otherwise.
	if numberOfFrames > 0 && t.currentScale > 0 {
		// The scale grows geometrically, so the frame scale belongs in the
		// exponent rather than as a factor on the result.
		stepped := numeric.E.Pow(t.targetScale.Log() / numberOfFrames * scale)

		// Land on the target rather than past it.
		if next := t.currentScale * stepped; (t.targetScale >= 1 && next <= t.targetScale) ||
			(t.targetScale < 1 && next >= t.targetScale) {

			sizeFactor = stepped
		}
	}

	t.size, t.position = t.size.Resize(sizeFactor, t.position)
	t.size.Scale = 1
	t.currentScale *= sizeFactor
}

// SetAnimationDuration sets the animation duration of the transition.
func (t *SizeTransition) SetAnimationDuration(duration time.Duration) *SizeTransition {
	t.animationDuration = duration
	return t
}

// SetImmutable sets the immutable property of the transition.
func (t *SizeTransition) SetImmutable(immutable bool) *SizeTransition {
	t.immutable = immutable
	return t
}

// SetPosition sets the position of the transition.
func (t *SizeTransition) SetPosition(position numeric.Position) *SizeTransition {
	t.position = position
	return t
}

// SetScale sets the target scale of the transition.
// Current scale is used as the starting point of the transition.
func (t *SizeTransition) SetScale(scale numeric.Number) *SizeTransition {
	if t.immutable {
		return t
	}

	if !numeric.Equal(t.targetScale, scale, 1e-9) {
		t.size, t.position = t.size.Resize(1/t.currentScale, t.position)
		t.size.Scale = 1
		t.targetScale, t.currentScale = scale, 1
	}

	return t
}

// SetPosition sets the position of the transition.
func (t *SizeTransition) SetSize(size numeric.Size) *SizeTransition {
	t.size = size
	return t
}

// SetTransitionEnd sets the transition end function of the transition.
func (t *SizeTransition) SetTransitionEnd(transitionEnd func(*SizeTransition)) *SizeTransition {
	t.transitionEnd = transitionEnd
	return t
}

// Position returns the current position of the transition.
func (t *SizeTransition) Position() numeric.Position { return t.position }

// Size returns the current size of the transition.
func (t *SizeTransition) Size() numeric.Size { return t.size }

// InitialSizeTransition initializes a size transition with the size, and position.
func InitialSizeTransition(size numeric.Size, position numeric.Position) *SizeTransition {
	t := SizeTransition{
		animationDuration: config.Config.Control.AnimationDuration,
		size:              size,
		currentScale:      1,
		targetScale:       1,
		position:          position,
	}

	t.size.Scale = 1

	return &t
}
