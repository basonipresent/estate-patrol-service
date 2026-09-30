// Package treestats computes tree height statistics from a height histogram.
package treestats

import (
	"cmp"
	"slices"
)

// HeightCount is how many trees of one height an estate has.
type HeightCount struct {
	Height int
	Count  int
}

type Stats struct {
	Count  int
	Min    int
	Max    int
	Median float64
}

// FromHistogram computes stats from per-height counts (each Count > 0).
// Heights are 1..30, so the input has at most 30 entries whatever the tree count.
// With an even count the median is the mean of the two middle heights.
func FromHistogram(hist []HeightCount) Stats {
	if len(hist) == 0 {
		return Stats{}
	}

	sorted := slices.Clone(hist)
	slices.SortFunc(sorted, func(a, b HeightCount) int {
		return cmp.Compare(a.Height, b.Height)
	})

	n := 0
	for _, b := range sorted {
		n += b.Count
	}

	// The median sits at the middle index, or between the two middle ones.
	lo, hi := (n-1)/2, n/2
	var loHeight, hiHeight, seen int
	for _, b := range sorted {
		if lo >= seen && lo < seen+b.Count {
			loHeight = b.Height
		}
		if hi >= seen && hi < seen+b.Count {
			hiHeight = b.Height
		}
		seen += b.Count
	}

	return Stats{
		Count:  n,
		Min:    sorted[0].Height,
		Max:    sorted[len(sorted)-1].Height,
		Median: float64(loHeight+hiHeight) / 2,
	}
}
