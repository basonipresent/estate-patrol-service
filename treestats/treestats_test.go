package treestats

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromHistogram(t *testing.T) {
	tests := []struct {
		name string
		hist []HeightCount
		want Stats
	}{
		{name: "no trees", hist: nil, want: Stats{}},
		{name: "single tree", hist: []HeightCount{{Height: 7, Count: 1}}, want: Stats{Count: 1, Min: 7, Max: 7, Median: 7}},
		{
			name: "spec example 5,3,4",
			hist: []HeightCount{{Height: 5, Count: 1}, {Height: 3, Count: 1}, {Height: 4, Count: 1}},
			want: Stats{Count: 3, Min: 3, Max: 5, Median: 4},
		},
		{
			name: "even count averages the middle two",
			hist: []HeightCount{{Height: 3, Count: 1}, {Height: 4, Count: 1}},
			want: Stats{Count: 2, Min: 3, Max: 4, Median: 3.5},
		},
		{
			name: "all the same height",
			hist: []HeightCount{{Height: 12, Count: 4}},
			want: Stats{Count: 4, Min: 12, Max: 12, Median: 12},
		},
		{
			name: "median inside a repeated height",
			hist: []HeightCount{{Height: 30, Count: 1}, {Height: 1, Count: 2}},
			want: Stats{Count: 3, Min: 1, Max: 30, Median: 1},
		},
		{
			name: "middle two in different buckets",
			hist: []HeightCount{{Height: 9, Count: 2}, {Height: 5, Count: 2}},
			want: Stats{Count: 4, Min: 5, Max: 9, Median: 7},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, FromHistogram(tc.hist))
		})
	}
}

func TestFromHistogramDoesNotModifyInput(t *testing.T) {
	hist := []HeightCount{{Height: 9, Count: 1}, {Height: 2, Count: 3}}
	before := slices.Clone(hist)

	FromHistogram(hist)

	require.Equal(t, before, hist)
}

// Compare against the textbook median of the expanded, sorted list.
func TestMedianMatchesSorting(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		var hist []HeightCount
		var all []int
		for h := 1; h <= 30; h++ {
			if r.Intn(3) == 0 {
				c := r.Intn(5) + 1
				hist = append(hist, HeightCount{Height: h, Count: c})
				for j := 0; j < c; j++ {
					all = append(all, h)
				}
			}
		}
		if len(all) == 0 {
			continue
		}
		r.Shuffle(len(hist), func(i, j int) { hist[i], hist[j] = hist[j], hist[i] })

		slices.Sort(all)
		n := len(all)
		want := float64(all[(n-1)/2]+all[n/2]) / 2

		require.Equal(t, want, FromHistogram(hist).Median, "hist=%v", hist)
	}
}
