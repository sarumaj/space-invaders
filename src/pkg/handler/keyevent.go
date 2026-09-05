package handler

const (
	ArrowDown  keyBinding = "ArrowDown"  // ArrowDown represents the down arrow key.
	ArrowLeft  keyBinding = "ArrowLeft"  // ArrowLeft represents the left arrow key.
	ArrowRight keyBinding = "ArrowRight" // ArrowRight represents the right arrow key.
	ArrowUp    keyBinding = "ArrowUp"    // ArrowUp represents the up arrow key.
	Escape     keyBinding = "Escape"     // Escape represents the escape key.
	KeyA       keyBinding = "KeyA"       // KeyA represents the A key.
	KeyD       keyBinding = "KeyD"       // KeyD represents the D key.
	KeyP       keyBinding = "KeyP"       // KeyP represents the P key.
	KeyS       keyBinding = "KeyS"       // KeyS represents the S key.
	KeyW       keyBinding = "KeyW"       // KeyW represents the W key.
	Pause      keyBinding = "Pause"      // Pause represents the pause key.
	Space      keyBinding = "Space"      // Space represents the space key.
)

const (
	actionNone      action = iota // actionNone is the action of a key that does nothing.
	actionMoveLeft                // actionMoveLeft moves the spaceship to the left.
	actionMoveRight               // actionMoveRight moves the spaceship to the right.
	actionMoveUp                  // actionMoveUp moves the spaceship up.
	actionMoveDown                // actionMoveDown moves the spaceship down.
	actionFire                    // actionFire fires the cannons.
	actionPause                   // actionPause pauses the game.
)

// heldActions lists the actions that repeat while their key stays down, in the
// order they are applied each frame. The order is fixed on purpose: ranging over
// the map of held keys made the outcome depend on Go's randomized map iteration,
// which is observable as soon as one action can invert another, as the hijacked
// spaceship state does.
var heldActions = [...]action{actionMoveLeft, actionMoveRight, actionMoveUp, actionMoveDown, actionFire}

// action represents what a key does, independently of which key was pressed.
type action int

// keyBinding represents a key binding.
// It holds the value of the KeyboardEvent code property, which identifies the
// physical key and is therefore independent of the keyboard layout.
type keyBinding string

// Action returns the action bound to the key.
// Several keys share an action so that WASD and the arrow block are
// interchangeable, and so that pausing does not depend on the Pause key, which
// most laptop and Mac keyboards do not have.
func (k keyBinding) Action() action {
	switch k {
	case ArrowLeft, KeyA:
		return actionMoveLeft

	case ArrowRight, KeyD:
		return actionMoveRight

	case ArrowUp, KeyW:
		return actionMoveUp

	case ArrowDown, KeyS:
		return actionMoveDown

	case Space:
		return actionFire

	case Pause, Escape, KeyP:
		return actionPause

	default:
		return actionNone
	}
}

// keyEvent represents a key event.
type keyEvent struct {
	Key     keyBinding
	Pressed bool
}
