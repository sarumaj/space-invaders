//go:build !js || !wasm

package handler

import (
	"sync"
	"time"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// frameSource is the shared frame channel, and frameSourceOnce guards its
// construction.
var (
	frameSource     <-chan numeric.Number
	frameSourceOnce sync.Once
)

// ask is a method that asks the user for input.
func (h *handler) ask() {}

// frames returns a channel that yields one value per frame, carrying how long
// that frame lasted expressed in nominal frames. Off the browser there is no
// display to synchronise with, so the frames are paced by a ticker.
func frames() <-chan numeric.Number {
	// The game loop is restarted after every game over, so the ticker below is
	// built once and shared.
	frameSourceOnce.Do(func() { frameSource = newFrameSource() })

	return frameSource
}

// newFrameSource starts the ticker feeding the game loop.
func newFrameSource() <-chan numeric.Number {
	out := make(chan numeric.Number, 1)
	interval := time.Second / time.Duration(config.Config.Control.DesiredFramesPerSecondRate)

	go func() {
		for range time.Tick(interval) {
			select {
			case out <- 1:
			default:
			}
		}
	}()

	return out
}

// registerEventHandlers is a method that registers the event listeners.
func (h *handler) registerEventHandlers() func() {
	h.once.Do(func() {})
	return func() {}
}
