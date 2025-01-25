package utils

func CompareUnorderedStringSlices(expected, actual []string) bool {
	// If the slices have different lengths, they cannot be equal
	if len(expected) != len(actual) {
		return false
	}

	// Create maps to count occurrences of each string
	expectedCounts := make(map[string]int)
	actualCounts := make(map[string]int)

	// Count occurrences in the expected slice
	for _, v := range expected {
		expectedCounts[v]++
	}

	// Count occurrences in the actual slice
	for _, v := range actual {
		actualCounts[v]++
	}

	// Compare the counts
	for key, count := range expectedCounts {
		if actualCounts[key] != count {
			return false
		}
	}
	return true
}
