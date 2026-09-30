package droneplan

import (
	"cmp"
	"slices"
)

// plotSize is the side of one plot in meters.
const plotSize = 10

// Tree stands on plot (X, Y), 1-based from the south-west corner.
type Tree struct {
	X      int
	Y      int
	Height int
}

// Distance returns the total meters flown to patrol the estate, including
// take-off and landing. It is O(T log T) in the number of trees and never
// iterates over empty plots. Trees must be in bounds and on distinct plots.
func Distance(length, width int, trees []Tree) int64 {
	plots := int64(length) * int64(width)
	horizontal := plotSize * (plots - 1)
	vertical := int64(2) // 1 m take-off + 1 m landing

	// Virtual ground stops at both ends remove the first/last-plot special cases.
	stops := make([]stop, 0, len(trees)+2)
	stops = append(stops, stop{pos: -1})
	for _, t := range trees {
		stops = append(stops, stop{pos: pathIndex(length, t.X, t.Y), height: t.Height})
	}
	stops = append(stops, stop{pos: plots})
	slices.SortFunc(stops[1:len(stops)-1], func(a, b stop) int {
		return cmp.Compare(a.pos, b.pos)
	})

	for i := 1; i < len(stops); i++ {
		vertical += heightChange(stops[i-1], stops[i])
	}

	return horizontal + vertical
}

type stop struct {
	pos    int64
	height int // 0 for ground
}

func heightChange(prev, next stop) int64 {
	if next.pos == prev.pos+1 {
		return int64(abs(next.height - prev.height))
	}
	// Empty plots in between: down to ground level and back up.
	return int64(prev.height + next.height)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
