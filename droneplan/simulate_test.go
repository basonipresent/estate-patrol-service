package droneplan

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func simulate(length, width int, trees []Tree) int64 {
	heights := make(map[int64]int, len(trees))
	for _, t := range trees {
		heights[pathIndex(length, t.X, t.Y)] = t.Height
	}

	plots := int64(length) * int64(width)
	var dist int64
	alt := 0 // on the ground
	for p := int64(0); p < plots; p++ {
		target := heights[p] + 1
		dist += int64(abs(target - alt))
		alt = target
		if p < plots-1 {
			dist += plotSize
		}
	}

	return dist + int64(alt) // land from the last plot
}

func TestDistanceMatchesSimulation(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 5000; i++ {
		length, width := r.Intn(8)+1, r.Intn(8)+1
		var trees []Tree
		for y := 1; y <= width; y++ {
			for x := 1; x <= length; x++ {
				if r.Intn(2) == 0 {
					trees = append(trees, Tree{X: x, Y: y, Height: r.Intn(30) + 1})
				}
			}
		}
		// Shuffle so Distance never sees the trees already in path order.
		r.Shuffle(len(trees), func(i, j int) {
			trees[i], trees[j] = trees[j], trees[i]
		})

		require.Equal(t, simulate(length, width, trees), Distance(length, width, trees),
			"length=%d width=%d trees=%v", length, width, trees)
	}
}
