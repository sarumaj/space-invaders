package config

import "testing"

// TestShieldGauge pins the two regimes of the shield row: exact while a pip can
// be spared per charge, and proportional once capacity outgrows the row.
func TestShieldGauge(t *testing.T) {
	for _, tt := range []struct {
		name           string
		charge         int
		capacity       int
		filled, shown  int
		wantSummarized bool
	}{
		{name: "no shield", charge: 0, capacity: 0, filled: 0, shown: 0},
		{name: "empty", charge: 0, capacity: 3, filled: 0, shown: 3},
		{name: "partial", charge: 2, capacity: 3, filled: 2, shown: 3},
		{name: "full", charge: 3, capacity: 3, filled: 3, shown: 3},
		{name: "exactly at the limit", charge: 5, capacity: 8, filled: 5, shown: 8},

		// The case that sent this back for a fix: eight pips filled by absolute
		// charge read as a full shield at 22 of 71.
		{name: "summarized low", charge: 22, capacity: 71, filled: 2, shown: 8, wantSummarized: true},
		{name: "summarized half", charge: 36, capacity: 71, filled: 4, shown: 8, wantSummarized: true},
		{name: "summarized full", charge: 71, capacity: 71, filled: 8, shown: 8, wantSummarized: true},

		// Neither end may lie: a shield with something in it never reads empty,
		// and one with anything missing never reads full.
		{name: "summarized almost empty", charge: 1, capacity: 71, filled: 1, shown: 8, wantSummarized: true},
		{name: "summarized almost full", charge: 70, capacity: 71, filled: 7, shown: 8, wantSummarized: true},

		// Charge outside the capacity is clamped rather than trusted.
		{name: "charge over capacity", charge: 99, capacity: 4, filled: 4, shown: 4},
		{name: "negative charge", charge: -3, capacity: 4, filled: 0, shown: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := HUD{ShieldCharge: tt.charge, ShieldCapacity: tt.capacity}

			filled, shown := state.ShieldGauge()
			if filled != tt.filled || shown != tt.shown {
				t.Errorf("HUD{%d/%d}.ShieldGauge() = %d/%d, want %d/%d",
					tt.charge, tt.capacity, filled, shown, tt.filled, tt.shown)
			}

			if filled > shown {
				t.Errorf("HUD{%d/%d}.ShieldGauge() filled %d of only %d pips",
					tt.charge, tt.capacity, filled, shown)
			}

			if got := state.ShieldSummarized(); got != tt.wantSummarized {
				t.Errorf("HUD{%d/%d}.ShieldSummarized() = %v, want %v",
					tt.charge, tt.capacity, got, tt.wantSummarized)
			}
		})
	}
}
