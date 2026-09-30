// Package droneplan computes drone patrol distances. The drone flies a
// serpentine (odd rows east, even rows west), so every step is 10 m and the
// estate can be treated as a single line of plots.
package droneplan

// pathIndex returns the 0-based position of plot (x, y) on the patrol path.
func pathIndex(length, x, y int) int64 {
	row := int64(y - 1)
	rowStart := row * int64(length)
	if row%2 == 0 {
		// Eastbound
		return rowStart + int64(x-1)
	}
	// Westbound
	return rowStart + int64(length-x)
}

// plotAt is the inverse of pathIndex.
func plotAt(length int, p int64) (x, y int) {
	l := int64(length)
	row, offset := p/l, p%l
	y = int(row) + 1
	if row%2 == 0 {
		// Eastbound
		return int(offset) + 1, y
	}
	// Westbound
	return int(l - offset), y
}
