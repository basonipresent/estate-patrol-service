package droneplan

import (
	"cmp"
	"slices"
)

// Rest returns the plot where the drone lands after flying maxDistance meters.
// Mid-move it lands on the plot it just left; past the end, on the last plot.
func Rest(length, width int, trees []Tree, maxDistance int64) (x, y int) {
	plots := int64(length) * int64(width)
	stops := make([]stop, 0, len(trees))
	for _, t := range trees {
		stops = append(stops, stop{pos: pathIndex(length, t.X, t.Y), height: t.Height})
	}
	slices.SortFunc(stops, func(a, b stop) int { return cmp.Compare(a.pos, b.pos) })

	next := 0 // index of the first tree not yet reached
	heightAt := func(p int64) int {
		if next < len(stops) && stops[next].pos == p {
			next++
			return stops[next-1].height
		}
		return 0
	}

	p := int64(0)
	cur := heightAt(0)
	flown := int64(cur + 1) // take-off
	if maxDistance <= flown {
		return plotAt(length, 0)
	}

	for p < plots-1 {
		nextTree := plots
		if next < len(stops) {
			nextTree = stops[next].pos
		}

		// Run of empty plots: 10 m hops at 1 m altitude, skipped arithmetically.
		if cur == 0 && nextTree > p+1 {
			end := min(nextTree-1, plots-1)
			if maxDistance < flown+plotSize*(end-p) {
				return plotAt(length, p+(maxDistance-flown)/plotSize)
			}
			flown += plotSize * (end - p)
			p = end
			continue
		}

		h := heightAt(p + 1)
		if h > cur { // climb before moving
			flown += int64(h - cur)
			if maxDistance <= flown {
				return plotAt(length, p)
			}
		}
		flown += plotSize
		if maxDistance < flown {
			return plotAt(length, p)
		}
		if maxDistance == flown {
			return plotAt(length, p+1)
		}
		if h < cur { // descend after moving
			flown += int64(cur - h)
			if maxDistance <= flown {
				return plotAt(length, p+1)
			}
		}
		p, cur = p+1, h
	}
	return plotAt(length, plots-1)
}
