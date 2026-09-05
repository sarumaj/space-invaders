package config

import "math"

// maximumShieldPips is how many pips the shield gauge draws. Shield capacity
// grows by one on every level up and is never bounded, so the row cannot be one
// pip per charge: left to grow it pushed the rest of the display off the canvas.
const maximumShieldPips = 8

// HUD carries the values the on-screen display shows.
// The player used to have to read the scrolling message log to learn their own
// score, and the only number permanently on screen was the frame rate.
type HUD struct {
	Score          int
	Level          int
	Cannons        int
	ShieldCharge   int
	ShieldCapacity int
	Experience     float64 // Progress towards the next level, from 0 to 1
}

// ShieldGauge returns how many pips of how many the shield row should fill.
//
// Below the pip limit there is one pip per charge and the gauge is exact. Above
// it the pips become a proportion of the capacity, because a row of eight pips
// filled by absolute charge reads as a full shield from the very first level at
// which capacity outgrows the row.
//
// The two ends are pinned: any charge at all keeps one pip lit, and any shortfall
// leaves one pip dark, so the gauge never reads as empty or as full while it is
// neither.
func (state HUD) ShieldGauge() (filled, shown int) {
	if state.ShieldCapacity <= 0 {
		return 0, 0
	}

	charge := min(max(state.ShieldCharge, 0), state.ShieldCapacity)
	shown = min(state.ShieldCapacity, maximumShieldPips)

	if shown == state.ShieldCapacity {
		return charge, shown
	}

	filled = int(math.Round(float64(shown) * float64(charge) / float64(state.ShieldCapacity)))

	if charge > 0 {
		filled = max(filled, 1)
	}

	if charge < state.ShieldCapacity {
		filled = min(filled, shown-1)
	}

	return filled, shown
}

// ShieldSummarized reports whether the gauge is an approximation, in which case
// the exact figures have to be spelled out next to it.
func (state HUD) ShieldSummarized() bool {
	return state.ShieldCapacity > maximumShieldPips
}
