package graphics

import (
	"testing"
	"time"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

func TestTransition(t *testing.T) {
	transition := InitialSizeTransition(numeric.Size{Width: 1, Height: 1}, numeric.Position{X: 0, Y: 0})

	transition.SetScale(2)

	ticker := time.NewTicker(time.Second / 60)
	timer := time.NewTimer(config.Config.Control.AnimationDuration)
	defer ticker.Stop()
	defer timer.Stop()

test:
	for {
		select {
		case <-ticker.C:
			transition.Interpolate(1)
			t.Logf("Size: %v, Position: %v", transition.Size(), transition.Position())

		case <-timer.C:
			break test
		}
	}

	if !numeric.Equal(transition.Size(), numeric.Size{Width: 2, Height: 2, Scale: 2}, 1e-9) {
		t.Errorf("Size: got %v, want %v", transition.Size(), numeric.Size{Width: 2, Height: 2})
	}

	if !numeric.Equal(transition.Position(), numeric.Position{X: -0.5, Y: -0.5}, 1e-9) {
		t.Errorf("Position: got %v, want %v", transition.Position(), numeric.Position{X: 0, Y: 0})
	}
}
