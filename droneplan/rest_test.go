package droneplan

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRest(t *testing.T) {
	// Spec example, 5x1 with trees 5, 3, 4 on x = 2..4. Flight log:
	//   take-off 1 → 1 | climb 5 → 6 | move → 16 at (2,1)
	//   move → 26 at (3,1) | descend 2 → 28 | climb 1 → 29 | move → 39 at (4,1)
	//   move → 49 at (5,1) | descend 4 → 53 | land 1 → 54
	spec := []Tree{{X: 2, Y: 1, Height: 5}, {X: 3, Y: 1, Height: 3}, {X: 4, Y: 1, Height: 4}}

	tests := []struct {
		name          string
		length, width int
		trees         []Tree
		maxDistance   int64
		wantX, wantY  int
	}{
		{name: "end of take-off", length: 5, width: 1, trees: spec, maxDistance: 1, wantX: 1, wantY: 1},
		{name: "mid climb lands where it is", length: 5, width: 1, trees: spec, maxDistance: 3, wantX: 1, wantY: 1},
		{name: "mid move lands on the plot it left", length: 5, width: 1, trees: spec, maxDistance: 10, wantX: 1, wantY: 1},
		{name: "exactly on arrival", length: 5, width: 1, trees: spec, maxDistance: 16, wantX: 2, wantY: 1},
		{name: "arrived before descending", length: 5, width: 1, trees: spec, maxDistance: 26, wantX: 3, wantY: 1},
		{name: "mid descent lands on the new plot", length: 5, width: 1, trees: spec, maxDistance: 27, wantX: 3, wantY: 1},
		{name: "during final landing", length: 5, width: 1, trees: spec, maxDistance: 53, wantX: 5, wantY: 1},
		{name: "battery covers the whole plan", length: 5, width: 1, trees: spec, maxDistance: 1000, wantX: 5, wantY: 1},
		{name: "empty run, mid hop", length: 3, width: 2, maxDistance: 25, wantX: 3, wantY: 1},
		{name: "empty run, across the row turn", length: 3, width: 2, maxDistance: 31, wantX: 3, wantY: 2},
		{name: "largest estate, second row", length: 50000, width: 50000, maxDistance: 1 + 10*50000, wantX: 50000, wantY: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x, y := Rest(tc.length, tc.width, tc.trees, tc.maxDistance)
			require.Equal(t, tc.wantX, x, "x")
			require.Equal(t, tc.wantY, y, "y")
		})
	}
}

// simulateRest replays the flight segment by segment. O(N), only for checking Rest.
func simulateRest(length, width int, trees []Tree, maxDistance int64) (x, y int) {
	heights := make(map[int64]int, len(trees))
	for _, t := range trees {
		heights[pathIndex(length, t.X, t.Y)] = t.Height
	}
	alt := func(p int64) int64 { return int64(heights[p]) + 1 }

	// Each segment: its cost, the plot if the battery dies during it, and the plot at its end.
	type segment struct{ cost, during, end int64 }
	last := int64(length)*int64(width) - 1
	segs := []segment{{cost: alt(0), during: 0, end: 0}}
	for p := int64(0); p < last; p++ {
		a, b := alt(p), alt(p+1)
		if b > a {
			segs = append(segs, segment{cost: b - a, during: p, end: p})
		}
		segs = append(segs, segment{cost: plotSize, during: p, end: p + 1})
		if b < a {
			segs = append(segs, segment{cost: a - b, during: p + 1, end: p + 1})
		}
	}
	segs = append(segs, segment{cost: alt(last), during: last, end: last})

	var flown int64
	for _, s := range segs {
		if maxDistance < flown+s.cost {
			return plotAt(length, s.during)
		}
		flown += s.cost
		if maxDistance == flown {
			return plotAt(length, s.end)
		}
	}
	return plotAt(length, last)
}

func TestRestMatchesSimulation(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		length, width := r.Intn(6)+1, r.Intn(6)+1
		var trees []Tree
		for y := 1; y <= width; y++ {
			for x := 1; x <= length; x++ {
				if r.Intn(3) == 0 {
					trees = append(trees, Tree{X: x, Y: y, Height: r.Intn(30) + 1})
				}
			}
		}
		r.Shuffle(len(trees), func(i, j int) { trees[i], trees[j] = trees[j], trees[i] })

		// Every possible battery size, including past the end of the plan.
		total := Distance(length, width, trees)
		for maxDistance := int64(1); maxDistance <= total+3; maxDistance++ {
			wantX, wantY := simulateRest(length, width, trees, maxDistance)
			gotX, gotY := Rest(length, width, trees, maxDistance)
			require.Equal(t, [2]int{wantX, wantY}, [2]int{gotX, gotY},
				"%dx%d trees=%v max_distance=%d", length, width, trees, maxDistance)
		}
	}
}
