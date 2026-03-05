package tui

type LayoutEntry struct {
	Weight  int
	MinSize int
}

// DistributeSpace splits total pixels among entries proportionally by
// weight, guaranteeing each entry at least MinSize. Rounding remainders
// go to the first entries. If total < sum of MinSizes, every entry still
// gets its MinSize (caller handles the overflow).
func DistributeSpace(total int, entries []LayoutEntry) []int {
	n := len(entries)
	if n == 0 {
		return nil
	}

	sizes := make([]int, n)

	minTotal := 0
	for i, e := range entries {
		sizes[i] = e.MinSize
		minTotal += e.MinSize
	}

	remaining := total - minTotal
	if remaining <= 0 {
		return sizes
	}

	totalWeight := 0
	for _, e := range entries {
		totalWeight += e.Weight
	}
	if totalWeight == 0 {
		return sizes
	}

	distributed := 0
	for i, e := range entries {
		share := remaining * e.Weight / totalWeight
		sizes[i] += share
		distributed += share
	}

	leftover := remaining - distributed
	for i := 0; leftover > 0 && i < n; i++ {
		sizes[i]++
		leftover--
	}

	return sizes
}
