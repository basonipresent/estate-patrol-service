package droneplan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathIndex(t *testing.T) {
	tests := []struct {
		name   string
		length int
		x, y   int
		want   int64
	}{
		{name: "single plot", length: 1, x: 1, y: 1, want: 0},
		{name: "eastbound row start", length: 5, x: 1, y: 1, want: 0},
		{name: "eastbound row end", length: 5, x: 5, y: 1, want: 4},
		{name: "single column row 2", length: 1, x: 1, y: 2, want: 1},
		{name: "single column row 3", length: 1, x: 1, y: 3, want: 2},
		{name: "row turn before", length: 3, x: 3, y: 1, want: 2},
		{name: "row turn after (westbound start)", length: 3, x: 3, y: 2, want: 3},
		{name: "westbound row end", length: 3, x: 1, y: 2, want: 5},
		{name: "third row eastbound again", length: 3, x: 1, y: 3, want: 6},
		{name: "largest estate last plot", length: 50000, x: 1, y: 50000, want: 2_499_999_999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, pathIndex(tt.length, tt.x, tt.y))

			x, y := plotAt(tt.length, tt.want)
			require.Equal(t, tt.x, x, "x")
			require.Equal(t, tt.y, y, "y")
		})
	}
}

// Every plot maps to a unique index in [0, N), and plotAt maps it back.
func TestPathIndexRoundTrip(t *testing.T) {
	sizes := []struct{ length, width int }{
		{1, 1}, {1, 4}, {4, 1}, {4, 3}, {3, 4}, {5, 5},
	}

	for _, s := range sizes {
		n := s.length * s.width
		seen := make([]bool, n)
		for y := 1; y <= s.width; y++ {
			for x := 1; x <= s.length; x++ {
				p := pathIndex(s.length, x, y)
				require.GreaterOrEqual(t, p, int64(0))
				require.Less(t, p, int64(n))
				require.False(t, seen[p], "index %d assigned twice (%dx%d)", p, s.length, s.width)
				seen[p] = true

				gotX, gotY := plotAt(s.length, p)
				require.Equal(t, x, gotX)
				require.Equal(t, y, gotY)
			}
		}
		for p, ok := range seen {
			require.True(t, ok, "index %d unassigned (%dx%d)", p, s.length, s.width)
		}
	}
}
