package graphics

import (
	"testing"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// nominalFrames is how many nominal frames the configured animation duration
// lasts, which is what a transition driven at a scale of 1 must take.
func nominalFrames() int {
	return int(config.Config.Control.AnimationDuration.Seconds() *
		config.Config.Control.DesiredFramesPerSecondRate)
}

// runUntil drives step until done reports true, and returns the number of calls
// it took. The cap keeps a transition that never converges from hanging the test.
func runUntil(step func(), done func() bool) int {
	for frames := 0; frames < 100_000; frames++ {
		if done() {
			return frames
		}

		step()
	}

	return -1
}

// TestColorTransitionHonoursFrameScale checks that a colour transition takes the
// same wall-clock time at any refresh rate: half-length frames need twice as many
// of them, double-length frames half as many. It also pins that the transition
// lands exactly on the target, which a fractional scale would otherwise overshoot.
func TestColorTransitionHonoursFrameScale(t *testing.T) {
	for _, test := range []struct {
		scale numeric.Number
		want  int
	}{
		{scale: 0.5, want: nominalFrames() * 2},
		{scale: 1, want: nominalFrames()},
		{scale: 2, want: nominalFrames() / 2},
	} {
		transition := InitialColorTransition(Catalogue().Lavender())
		transition.SetColor(Catalogue().Crimson())

		frames := runUntil(
			func() { transition.Interpolate(test.scale) },
			func() bool { return transition.Gradient().Equal(Catalogue().Crimson()) },
		)

		if frames != test.want {
			t.Errorf("scale %v: took %d frames, want %d", test.scale, frames, test.want)
		}
	}
}

// TestSizeTransitionHonoursFrameScale is the size-transition counterpart.
func TestSizeTransitionHonoursFrameScale(t *testing.T) {
	target := numeric.Size{Width: 2, Height: 2, Scale: 2}

	for _, test := range []struct {
		scale numeric.Number
		want  int
	}{
		{scale: 0.5, want: nominalFrames() * 2},
		{scale: 1, want: nominalFrames()},
		{scale: 2, want: nominalFrames() / 2},
	} {
		transition := InitialSizeTransition(numeric.Size{Width: 1, Height: 1}, numeric.Position{})
		transition.SetScale(2)

		frames := runUntil(
			func() { transition.Interpolate(test.scale) },
			func() bool { return numeric.Equal(transition.Size(), target, 1e-9) },
		)

		if frames != test.want {
			t.Errorf("scale %v: took %d frames, want %d", test.scale, frames, test.want)
		}
	}
}

// TestTransitionEndFiresOnce pins that the completion callback is dropped after
// it runs. It used to be re-entered on every frame after the transition settled,
// so a callback that set a colour kept overwriting whatever was set after it.
func TestTransitionEndFiresOnce(t *testing.T) {
	var calls int

	transition := InitialColorTransition(Catalogue().Lavender())
	transition.SetColor(Catalogue().Crimson()).
		SetTransitionEnd(func(*ColorTransition) { calls++ })

	for i := 0; i < nominalFrames()*10; i++ {
		transition.Interpolate(1)
	}

	if calls != 1 {
		t.Errorf("transition end fired %d times, want 1", calls)
	}
}
