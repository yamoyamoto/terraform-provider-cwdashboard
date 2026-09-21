package provider

import (
	"math/rand"
	"testing"

	"github.com/tj/assert"
)

func widgets(sizes ...[2]int32) []CWDashboardBodyWidget {
	ws := make([]CWDashboardBodyWidget, 0, len(sizes))
	for _, s := range sizes {
		ws = append(ws, CWDashboardBodyWidget{Width: s[0], Height: s[1]})
	}

	return ws
}

func positions(ws []CWDashboardBodyWidget) [][2]int32 {
	ps := make([][2]int32, 0, len(ws))
	for _, w := range ws {
		ps = append(ps, [2]int32{w.X, w.Y})
	}

	return ps
}

// assertNoOverlap fails if any two widgets share a cell of the grid, or if a
// widget runs past the right edge.
func assertNoOverlap(t *testing.T, ws []CWDashboardBodyWidget) {
	t.Helper()

	occupied := map[[2]int32]int{}
	for i, w := range ws {
		assert.True(t, w.X >= 0 && w.X+w.Width <= MAX_WIDTH,
			"widget %d overflows the %d column grid: x=%d width=%d", i, MAX_WIDTH, w.X, w.Width)

		for x := w.X; x < w.X+w.Width; x++ {
			for y := w.Y; y < w.Y+w.Height; y++ {
				if other, taken := occupied[[2]int32{x, y}]; taken {
					t.Fatalf("widget %d overlaps widget %d at (%d, %d)", i, other, x, y)
				}
				occupied[[2]int32{x, y}] = i
			}
		}
	}
}

func TestLayoutWidgets(t *testing.T) {
	tests := []struct {
		name     string
		widgets  []CWDashboardBodyWidget
		expected [][2]int32
	}{
		{
			name:     "packs widgets of equal height left to right, then wraps",
			widgets:  widgets([2]int32{12, 6}, [2]int32{12, 6}, [2]int32{12, 6}),
			expected: [][2]int32{{0, 0}, {12, 0}, {0, 6}},
		},
		{
			name: "wraps below the previous row, not below the wrapping widget",
			// A full width heading followed by two half width graphs. The graphs
			// must start at y=2 (the bottom of the heading), not y=6.
			widgets:  widgets([2]int32{24, 2}, [2]int32{12, 6}, [2]int32{12, 6}),
			expected: [][2]int32{{0, 0}, {0, 2}, {12, 2}},
		},
		{
			name: "uses the tallest widget in the row as the row height",
			// Without tracking the row height the second widget would be placed at
			// y=4 and overlap the first one.
			widgets:  widgets([2]int32{18, 10}, [2]int32{12, 4}),
			expected: [][2]int32{{0, 0}, {0, 10}},
		},
		{
			name: "a full width widget always sits on a row of its own",
			// This is what makes a width 24 text widget behave as a section break.
			widgets:  widgets([2]int32{8, 6}, [2]int32{24, 2}, [2]int32{8, 6}),
			expected: [][2]int32{{0, 0}, {0, 6}, {0, 8}},
		},
		{
			name:     "fills a row exactly before wrapping",
			widgets:  widgets([2]int32{8, 6}, [2]int32{8, 6}, [2]int32{8, 6}, [2]int32{8, 6}),
			expected: [][2]int32{{0, 0}, {8, 0}, {16, 0}, {0, 6}},
		},
		{
			name:     "handles an empty dashboard",
			widgets:  widgets(),
			expected: [][2]int32{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			layoutWidgets(tc.widgets)

			assert.Equal(t, tc.expected, positions(tc.widgets))
			assertNoOverlap(t, tc.widgets)
		})
	}
}

func TestLayoutWidgets_NeverOverlaps(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		ws := make([]CWDashboardBodyWidget, 0, 100)
		for j := 0; j < 100; j++ {
			ws = append(ws, CWDashboardBodyWidget{
				Width:  int32(r.Intn(MAX_WIDTH) + 1),
				Height: int32(r.Intn(12) + 1),
			})
		}

		layoutWidgets(ws)
		assertNoOverlap(t, ws)
	}
}

func TestValidateWidgetSize(t *testing.T) {
	tests := []struct {
		name    string
		width   int32
		height  int32
		wantErr bool
	}{
		{name: "accepts a full width widget", width: 24, height: 6},
		{name: "accepts the smallest widget", width: 1, height: 1},
		{name: "rejects a widget wider than the grid", width: 25, height: 6, wantErr: true},
		{name: "rejects a zero width", width: 0, height: 6, wantErr: true},
		{name: "rejects a zero height", width: 8, height: 0, wantErr: true},
		{name: "rejects a height over the limit", width: 8, height: 1001, wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateWidgetSize(tc.width, tc.height)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
