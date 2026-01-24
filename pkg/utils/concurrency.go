package utils

import (
	"runtime"
)

// GetWorkers determines the optimal number of workers based on an optional workload count.
// Behavior:
//   - If count is not provided (nil), it selects a system-optimal default between 1 and 20.
//   - If count is provided and less than 5 (including 0), it returns count directly
//     to avoid worker overhead.
//   - For larger workloads, it scales the worker count using CPU*2 as a guideline.
//   - The worker count is capped at 20 to prevent excessive resource usage.
//
// This function is intended for primarily I/O-bound workloads such as file reading
// and template processing.
func GetWorkers(count *int) int {
	// Typically for I/O-bound tasks (like API calls), we can use more workers than CPUs
	// For CPU-bound tasks, we need stay close to the CPU count
	cpuCount := runtime.NumCPU()
	maxCpuCount := min(cpuCount*2, 20)

	if count == nil {
		return max(1, maxCpuCount)
	}

	if *count < 5 {
		return *count
	}

	return min(maxCpuCount, *count)
}
