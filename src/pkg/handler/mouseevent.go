package handler

import (
	"fmt"
	"sync"
	"time"

	"github.com/sarumaj/edu-space-invaders/src/pkg/numeric"
)

const (
	MouseButtonPrimary   mouseButton = iota // MouseButtonPrimary represents the primary mouse button.
	MouseButtonAuxiliary                    // MouseButtonAuxiliary represents the auxiliary mouse button.
	MouseButtonSecondary                    // MouseButtonSecondary represents the secondary mouse button.
)

// mouseButton represents the type of the mouse button.
type mouseButton int

// String returns the string representation of the mouse button.
func (b mouseButton) String() string {
	if b > 2 {
		return "Unknown"
	}

	return [...]string{"Primary", "Auxiliary", "Secondary"}[b]
}

// mouseEvent represents a mouse event.
type mouseEvent struct {
	mutex                                       *sync.Mutex
	StartPosition, CurrentPosition, EndPosition numeric.Position
	StartTime, EndTime                          time.Time
	Button                                      mouseButton
	Pressed                                     bool
	Type                                        mouseEventType
}

// ClickDuration returns the duration of the click.
func (m mouseEvent) ClickDuration() time.Duration {
	duration := m.EndTime.Sub(m.StartTime)
	if duration < 0 {
		duration = 0
	}

	return duration
}

// IsPressed reports whether a button is down, under the lock that guards the
// rest of the event. The listeners run on the JavaScript event loop while the
// game loop reads the same event, so an unguarded read of the field is a race.
func (m *mouseEvent) IsPressed() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.Pressed
}

// Send sends the mouse event to the specified channel.
func (m mouseEvent) Send(rcv chan<- mouseEvent) {
	rcv <- mouseEvent{
		StartPosition:   m.StartPosition,
		CurrentPosition: m.CurrentPosition,
		EndPosition:     m.EndPosition,
		StartTime:       m.StartTime,
		EndTime:         m.EndTime,
		Button:          m.Button,
		Pressed:         m.Pressed,
		Type:            m.Type,
	}
}

// SetButton sets the button of the mouse event.
func (m *mouseEvent) SetButton(b mouseButton) *mouseEvent {
	m.mutex.Lock()
	m.Button = b
	m.mutex.Unlock()

	return m
}

// SetCurrentPosition sets the current position of the mouse event.
func (m *mouseEvent) SetCurrentPosition(pos numeric.Position) *mouseEvent {
	m.mutex.Lock()
	m.CurrentPosition = pos
	m.mutex.Unlock()

	return m
}

// SetEndPosition sets the end position of the mouse event.
func (m *mouseEvent) SetEndPosition(pos numeric.Position) *mouseEvent {
	m.mutex.Lock()
	m.EndPosition = pos
	m.mutex.Unlock()

	return m
}

// SetEndTime sets the end time of the mouse event.
func (m *mouseEvent) SetEndTime(when time.Time) *mouseEvent {
	m.mutex.Lock()
	m.EndTime = when
	m.mutex.Unlock()

	return m
}

// SetPressed sets the pressed state of the mouse event.
func (m *mouseEvent) SetPressed(p bool) *mouseEvent {
	m.mutex.Lock()
	m.Pressed = p
	m.mutex.Unlock()

	return m
}

// SetStartPosition sets the start position of the mouse event.
func (m *mouseEvent) SetStartPosition(pos numeric.Position) *mouseEvent {
	m.mutex.Lock()
	m.StartPosition = pos
	m.mutex.Unlock()

	return m
}

// SetStartTime sets the start time of the mouse event.
func (m *mouseEvent) SetStartTime(when time.Time) *mouseEvent {
	m.mutex.Lock()
	m.StartTime = when
	m.mutex.Unlock()

	return m
}

// SetType sets the type of the mouse event.
func (m *mouseEvent) SetType(t mouseEventType) *mouseEvent {
	m.mutex.Lock()
	m.Type = t
	m.mutex.Unlock()

	return m
}

// Reset resets the mouse event.
func (m *mouseEvent) Reset() *mouseEvent {
	m.mutex.Lock()
	m.StartPosition = numeric.Position{}
	m.CurrentPosition = numeric.Position{}
	m.EndPosition = numeric.Position{}
	m.StartTime = time.Time{}
	m.EndTime = time.Time{}
	m.Button = 0
	m.Pressed = false
	m.mutex.Unlock()

	return m
}

// String returns the string representation of the mouse event.
func (m mouseEvent) String() string {
	return fmt.Sprintf("Mouse (Start: %s, End: %s, Duration: %s, Button: %s, Type: %s)",
		m.StartPosition, m.EndPosition, m.ClickDuration(), m.Button, m.Type)
}

const (
	MouseEventTypeUnknown mouseEventType = iota // MouseEventTypeUnknown represents an unknown mouse event type.
	MouseEventTypeDown                          // MouseEventTypeDown represents the start of the mouse event.
	MouseEventTypeMove                          // MouseEventTypeMove represents the move of the mouse event.
	MouseEventTypeUp                            // MouseEventTypeEnd represents the end of the mouse event.
)

// mouseEventType represents the type of the mouse event.
type mouseEventType int

// String returns the string representation of the mouse event type.
func (t mouseEventType) String() string {
	if t > MouseEventTypeUp {
		return "Unknown"
	}

	return [...]string{"Unknown", "Down", "Move", "Up"}[t]
}
