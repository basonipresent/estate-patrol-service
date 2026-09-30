package droneplan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistance(t *testing.T) {
	tests := []struct {
		name          string
		length, width int
		trees         []Tree
		want          int64
	}{
		{
			name:   "spec example: 40 m horizontal + 14 m vertical",
			length: 5, width: 1,
			trees: []Tree{{X: 2, Y: 1, Height: 5}, {X: 3, Y: 1, Height: 3}, {X: 4, Y: 1, Height: 4}},
			want:  54,
		},
		{
			name:   "starter API test: 40 m horizontal + 42 m vertical",
			length: 5, width: 1,
			trees: []Tree{{X: 2, Y: 1, Height: 10}, {X: 3, Y: 1, Height: 20}, {X: 4, Y: 1, Height: 10}},
			want:  82,
		},
		{name: "single empty plot: take-off and landing only", length: 1, width: 1, want: 2},
		{
			name:   "single plot with a tree",
			length: 1, width: 1,
			trees: []Tree{{X: 1, Y: 1, Height: 7}},
			want:  2*7 + 2,
		},
		{name: "empty estate", length: 4, width: 3, want: 10*(12-1) + 2},
		{
			name:   "trees on first and last plot",
			length: 3, width: 2,
			trees: []Tree{{X: 1, Y: 1, Height: 4}, {X: 1, Y: 2, Height: 6}},
			want:  10*5 + 2 + 4 + 4 + 6 + 6,
		},
		{
			name:   "row turn: (3,1) and (3,2) are neighbours on the path",
			length: 3, width: 2,
			trees: []Tree{{X: 3, Y: 1, Height: 5}, {X: 3, Y: 2, Height: 7}},
			want:  10*5 + 2 + 5 + 2 + 7,
		},
		{
			name:   "input order does not matter",
			length: 5, width: 1,
			trees: []Tree{{X: 4, Y: 1, Height: 4}, {X: 2, Y: 1, Height: 5}, {X: 3, Y: 1, Height: 3}},
			want:  54,
		},
		{
			name:   "largest estate, sparse trees: exact and int64-safe",
			length: 50000, width: 50000,
			trees: []Tree{{X: 1, Y: 1, Height: 30}, {X: 25000, Y: 25000, Height: 15}, {X: 1, Y: 50000, Height: 1}},
			want:  10*(50000*50000-1) + 2 + 30 + 30 + 15 + 15 + 1 + 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Distance(tc.length, tc.width, tc.trees))
		})
	}
}

func TestDistanceDoesNotModifyInput(t *testing.T) {
	trees := []Tree{{X: 4, Y: 1, Height: 4}, {X: 2, Y: 1, Height: 5}}
	before := append([]Tree(nil), trees...)

	Distance(5, 1, trees)

	require.Equal(t, before, trees)
}
