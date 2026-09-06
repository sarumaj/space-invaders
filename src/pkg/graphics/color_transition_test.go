package graphics

import (
	"testing"
	"time"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
)

func TestColorTransition(t *testing.T) {
	transition := InitialColorTransition(Catalogue().Lavender())

	transition.SetColor(Catalogue().Crimson())

	ticker := time.NewTicker(time.Second / 60)
	timer := time.NewTimer(config.Config.Control.AnimationDuration)
	defer ticker.Stop()
	defer timer.Stop()

test:
	for {
		select {
		case <-ticker.C:
			transition.Interpolate(1)
			t.Logf("CurrentGradient: %v", transition.Gradient())

		case <-timer.C:
			break test
		}
	}

	transition.Interpolate(1)
	if !transition.Gradient().Equal(Catalogue().Crimson()) {
		t.Errorf("CurrentGradient: got %v, want %v", transition.Gradient(), Catalogue().Crimson())
	}
}
