//go:build js && wasm

package handler

import (
	"sync"
	"syscall/js"
	"time"

	"github.com/sarumaj/edu-space-invaders/src/pkg/config"
	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

// maximumFrameScale caps how far a single frame may advance the simulation.
// A backgrounded tab resumes with an arbitrarily long gap, and letting objects
// travel that far in one step would tunnel them straight through every collision
// check.
const maximumFrameScale = 4

// frameSource is the shared animation frame channel, and frameSourceOnce guards
// its construction.
var (
	frameSource     <-chan numeric.Number
	frameSourceOnce sync.Once
)

// ask is a method that asks the user for input.
func (h *handler) ask() {
	if commandant := config.GlobalCall(
		"prompt",
		config.Execute(config.Config.MessageBox.Messages.Prompt),
		h.spaceship.Commandant,
	); commandant.Truthy() && commandant.String() != "" {

		h.spaceship.Commandant = commandant.String()
	}
}

// frames returns a channel that yields one value per animation frame, carrying
// how long that frame lasted expressed in nominal frames, so that motion can be
// scaled by it. Driving the loop from requestAnimationFrame keeps the simulation
// aligned with the display refresh; the fixed ticker this replaces was
// free-running against vsync, which showed as constant judder, and it made every
// speed in the configuration depend on the loop actually hitting its rate.
//
// The channel has room for a single frame and a frame is dropped rather than
// queued when the loop is still busy, so a slow frame costs one update instead of
// building a backlog the loop can never catch up with.
func frames() <-chan numeric.Number {
	// The game loop is restarted after every game over, and the animation frame
	// chain below never stops, so the source is built once and shared.
	frameSourceOnce.Do(func() { frameSource = newFrameSource() })

	return frameSource
}

// newFrameSource starts the animation frame chain feeding the game loop.
func newFrameSource() <-chan numeric.Number {
	const reportFramesEvery = 500 // milliseconds between frame-rate reports

	out := make(chan numeric.Number, 1)
	nominal := 1_000 / config.Config.Control.DesiredFramesPerSecondRate // milliseconds per nominal frame

	var previous, measuredSince float64
	var measuredFrames int

	// One js.Func, reused for the lifetime of the page. The frame watchdog this
	// replaces allocated a fresh one on every frame and released none of them,
	// leaking sixty entries of the callback registry per second.
	var step js.Func
	step = js.FuncOf(func(_ js.Value, p []js.Value) any {
		config.GlobalCall("requestAnimationFrame", step)

		now := p[0].Float()
		elapsed := now - previous
		if previous == 0 {
			elapsed, measuredSince = nominal, now
		}
		previous = now

		// Report the measured rate rather than acting on it: with motion scaled
		// by the frame time a slow frame no longer runs the game in slow motion,
		// so there is nothing left to suspend the game for.
		measuredFrames++
		if window := now - measuredSince; window >= reportFramesEvery {
			config.UpdateFPS(float64(measuredFrames) * 1_000 / window)
			measuredFrames, measuredSince = 0, now
		}

		select {
		case out <- numeric.Number(elapsed/nominal).Clamp(0, maximumFrameScale):
		default:
		}

		return nil
	})

	config.GlobalCall("requestAnimationFrame", step)

	return out
}

// registerEventHandlers is a method that registers the event listeners.
func (h *handler) registerEventHandlers() {
	h.once.Do(func() {
		config.GlobalSet("drawFunc", js.FuncOf(func(_ js.Value, _ []js.Value) any {
			h.draw(0) // A redraw after a resize must not advance the scrolling background.
			return nil
		}))

		config.GlobalSet("onlineFunc", js.FuncOf(func(_ js.Value, _ []js.Value) any {
			offline.Set(&h.ctx, false)
			return nil
		}))

		config.GlobalSet("offlineFunc", js.FuncOf(func(_ js.Value, _ []js.Value) any {
			offline.Set(&h.ctx, true)
			return nil
		}))

		// Every input method is registered on every device. Gating touch against
		// keyboard and mouse left a laptop with a touch screen — which reports a
		// non-zero maxTouchPoints, so it took the touch branch — with no keyboard
		// and no mouse at all, and a tablet with a keyboard attached with no keys.
		// The three now coexist, and a player may switch between them mid-game.
		globalTouchEvent := &touchEvent{mutex: &sync.Mutex{}}
		config.GlobalSet("touchstart", globalTouchEvent.touchStart(h.touchEvent))
		config.GlobalSet("touchmove", globalTouchEvent.touchMove(h.touchEvent))
		config.GlobalSet("touchend", globalTouchEvent.touchEnd(h.touchEvent))
		config.AddEventListenerToCanvas("touchstart", config.GlobalGet("touchstart"))
		config.AddEventListenerToCanvas("touchmove", config.GlobalGet("touchmove"))
		config.AddEventListenerToCanvas("touchend", config.GlobalGet("touchend"))
		// A gesture the system takes over — an incoming call, a palm on the
		// screen, the browser claiming it for a scroll — ends with touchcancel and
		// no touchend, which used to leave the spaceship firing on a finger that
		// was no longer there.
		config.AddEventListenerToCanvas("touchcancel", config.GlobalGet("touchend"))

		globalKeyMap := registeredKeys{
			ArrowDown:  true,
			ArrowLeft:  true,
			ArrowRight: true,
			ArrowUp:    true,
			Escape:     true,
			KeyA:       true,
			KeyD:       true,
			KeyP:       true,
			KeyS:       true,
			KeyW:       true,
			Pause:      true,
			Space:      true,
		}
		config.GlobalSet("keydown", globalKeyMap.keyDown(h.keyEvent))
		config.GlobalSet("keyup", globalKeyMap.keyUp(h.keyEvent))
		config.AddEventListener("keydown", config.GlobalGet("keydown"))
		config.AddEventListener("keyup", config.GlobalGet("keyup"))

		globalMouseEvent := &mouseEvent{mutex: &sync.Mutex{}}
		config.GlobalSet("mousedown", globalMouseEvent.mouseDown(h.mouseEvent))
		config.GlobalSet("mousemove", globalMouseEvent.mouseMove(h.mouseEvent))
		config.GlobalSet("mouseup", globalMouseEvent.mouseUp(h.mouseEvent))
		config.AddEventListenerToCanvas("contextmenu", config.GlobalGet("mousedown"))
		config.AddEventListenerToCanvas("mousedown", config.GlobalGet("mousedown"))
		config.AddEventListenerToCanvas("mousemove", config.GlobalGet("mousemove"))
		config.AddEventListenerToCanvas("mouseup", config.GlobalGet("mouseup"))
		// Releasing the button outside the canvas delivers no mouseup to it, so
		// the drag has to be ended when the pointer leaves instead.
		config.AddEventListenerToCanvas("mouseleave", config.GlobalGet("mouseup"))

		// The browser stops delivering key and pointer events the moment the page
		// loses focus, and never sends the matching release. Alt-tabbing with the
		// fire key down used to come back to a spaceship still firing.
		config.GlobalSet("release", js.FuncOf(func(_ js.Value, _ []js.Value) any {
			select {
			case h.releaseEvent <- struct{}{}:
			default: // A release is already queued; one is enough.
			}

			return nil
		}))
		config.AddEventListenerToWindow("blur", config.GlobalGet("release"))
		config.AddEventListener("visibilitychange", config.GlobalGet("release"))
	})
}

// registeredKeys represents a map of registered keys which are meant to be listened to.
type registeredKeys map[keyBinding]bool

// keyDown is a method that listens to the keydown event.
func (known registeredKeys) keyDown(rcv chan<- keyEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		key := keyBinding(p[0].Get("code").String())
		if !known[key] {
			return nil
		}

		p[0].Call("preventDefault")
		rcv <- keyEvent{
			Key:     key,
			Pressed: true,
		}

		return nil
	})
}

// keyUp is a method that listens to the keyup event.
func (known registeredKeys) keyUp(rcv chan<- keyEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		key := keyBinding(p[0].Get("code").String())
		if !known[key] {
			return nil
		}

		p[0].Call("preventDefault")
		rcv <- keyEvent{
			Key:     key,
			Pressed: false,
		}

		return nil
	})
}

// mouseDown is a method that listens to the mousedown or contextmenu event.
func (event *mouseEvent) mouseDown(rcv chan<- mouseEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		p[0].Call("preventDefault")

		canvasDimensions := config.CanvasBoundingBox()
		event.
			Reset().
			SetStartPosition(numeric.Position{
				X: numeric.Number(p[0].Get("clientX").Float() - canvasDimensions.BoxLeft),
				Y: numeric.Number(p[0].Get("clientY").Float() - canvasDimensions.BoxTop),
			}).
			SetButton(mouseButton(p[0].Get("button").Int())).
			SetStartTime(time.Now()).
			SetType(MouseEventTypeDown).
			SetPressed(true).
			Send(rcv)

		return nil
	})
}

// mouseMove is a method that listens to the mousemove event.
func (event *mouseEvent) mouseMove(rcv chan<- mouseEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		if !event.IsPressed() {
			return nil
		}

		p[0].Call("preventDefault")
		canvasDimensions := config.CanvasBoundingBox()
		_ = event.
			SetCurrentPosition(numeric.Position{
				X: numeric.Number(p[0].Get("clientX").Float() - canvasDimensions.BoxLeft),
				Y: numeric.Number(p[0].Get("clientY").Float() - canvasDimensions.BoxTop),
			}).
			SetType(MouseEventTypeMove)

		// Only the bitmask of buttons currently down is meaningful here: on a move
		// event `button` reports whichever button last changed state, not what is
		// being held, so pairing the two dropped drags that were perfectly valid.
		switch buttons := p[0].Get("buttons").Int(); {
		case buttons&1 != 0:
			_ = event.SetPressed(true).SetButton(MouseButtonPrimary)

		case buttons&2 != 0:
			_ = event.SetPressed(true).SetButton(MouseButtonSecondary)

		case buttons&4 != 0:
			_ = event.SetPressed(true).SetButton(MouseButtonAuxiliary)

		default:
			_ = event.SetPressed(false) // No buttons pressed
		}

		event.Send(rcv)
		return nil
	})
}

// mouseUp is a method that listens to the mouseup event.
func (event *mouseEvent) mouseUp(rcv chan<- mouseEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		p[0].Call("preventDefault")
		canvasDimensions := config.CanvasBoundingBox()
		event.
			SetEndPosition(numeric.Position{
				X: numeric.Number(p[0].Get("clientX").Float() - canvasDimensions.BoxLeft),
				Y: numeric.Number(p[0].Get("clientY").Float() - canvasDimensions.BoxTop),
			}).
			SetButton(mouseButton(p[0].Get("button").Int())).
			SetEndTime(time.Now()).
			SetPressed(false).
			SetType(MouseEventTypeUp).
			Send(rcv)

		return nil
	})
}

// touchEnd is a method that listens to the touchend event.
func (event *touchEvent) touchEnd(rcv chan<- touchEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		p[0].Call("preventDefault")
		changedTouches := p[0].Get("changedTouches")
		canvasDimensions := config.CanvasBoundingBox()
		event.
			SetEndPosition(numeric.Position{
				X: numeric.Number(changedTouches.Index(0).Get("clientX").Float() - canvasDimensions.BoxLeft),
				Y: numeric.Number(changedTouches.Index(0).Get("clientY").Float() - canvasDimensions.BoxTop),
			}).
			SetEndTime(time.Now()).
			SetMultiTap(multiTouch(p[0])).
			SetType(TouchTypeEnd).
			Send(rcv)

		return nil
	})
}

// touchMove is a method that listens to the touchmove event.
func (event *touchEvent) touchMove(rcv chan<- touchEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		p[0].Call("preventDefault")
		changedTouches := p[0].Get("changedTouches")
		canvasDimensions := config.CanvasBoundingBox()
		event.
			SetCurrentPosition(numeric.Position{
				X: numeric.Number(changedTouches.Index(0).Get("clientX").Float() - canvasDimensions.BoxLeft),
				Y: numeric.Number(changedTouches.Index(0).Get("clientY").Float() - canvasDimensions.BoxTop),
			}).
			SetMultiTap(multiTouch(p[0])).
			SetType(TouchTypeMove).
			Send(rcv)

		return nil
	})
}

// touchStart is a method that listens to the touchstart event.
func (event *touchEvent) touchStart(rcv chan<- touchEvent) js.Func {
	return js.FuncOf(func(_ js.Value, p []js.Value) any {
		p[0].Call("preventDefault")
		changedTouches := p[0].Get("changedTouches")
		canvasDimensions := config.CanvasBoundingBox()
		event.
			Reset().
			SetStartPosition(numeric.Position{
				X: numeric.Number(changedTouches.Index(0).Get("clientX").Float() - canvasDimensions.BoxLeft),
				Y: numeric.Number(changedTouches.Index(0).Get("clientY").Float() - canvasDimensions.BoxTop),
			}).
			SetStartTime(time.Now()).
			SetMultiTap(multiTouch(p[0])).
			SetType(TouchTypeStart).
			Send(rcv)

		return nil
	})
}

// multiTouch reports whether more than one finger is on the screen.
// The count has to come from the event's touches rather than its changedTouches:
// changedTouches carries only the fingers that moved or landed in this event, so
// putting two fingers down — which never happens in the same instant — reported
// one touch twice, and the two-finger tap that is meant to pause never fired.
func multiTouch(event js.Value) bool {
	touches := event.Get("touches")

	return touches.Truthy() && touches.Length() > 1
}
