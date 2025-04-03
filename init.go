// Package main initializes the project and runs the query
package main

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/features"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

/*
InitializeProject starts the project by creating envfolder and global.env file in it.
*/
func InitializeProject(env string) map[string]any {
	if err := envparser.CreateDefaultEnvs(nil); err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	envMap, err := envparser.GenerateSecretsMap(env)
	if err != nil {
		panic(err)
	}
	return envMap
}

// RunTasks manages the go tasks with a limited worker pool
func RunTasks(filePathList []string, secretsMap map[string]any, debug bool) {
	// Configuration parameters
	maxWorkers := calculateOptimalWorkerCount() // Dynamically determine worker count
	maxRetries := 3                             // Number of retries for failed tasks
	timeout := 60 * time.Second                 // Timeout for each API call

	var wg sync.WaitGroup
	taskChan := make(chan string, len(filePathList)) // Buffered channel for tasks

	// Fill the task channel with file paths
	for _, path := range filePathList {
		taskChan <- path
	}
	close(taskChan)

	// Create a pool of worker goroutines
	for i := range maxWorkers {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			for path := range taskChan {
				// Process each task with retry logic
				success := false
				var lastErr error

				for attempt := 0; attempt < maxRetries && !success; attempt++ {
					if attempt > 0 {
						// Exponential backoff for retries
						backoffDuration := time.Duration(1<<uint(attempt-1)) * time.Second
						utils.PrintWarning(fmt.Sprintf("Retrying %s (attempt %d/%d) after %v",
							path, attempt+1, maxRetries, backoffDuration))
						time.Sleep(backoffDuration)
					}

					// Create a context with timeout
					ctx, cancel := context.WithTimeout(context.Background(), timeout)

					// Use a channel to handle the task completion
					doneChan := make(chan struct{})
					errChan := make(chan error, 1)

					// Execute the task in a separate goroutine
					go func() {
						err := processTask(path, utils.CopyEnvMap(secretsMap), debug)
						if err != nil {
							errChan <- err
						} else {
							close(doneChan)
						}
					}()

					// Wait for either completion, error, or timeout
					select {
					case <-doneChan:
						success = true
						// No need to print success, it gets annoying
					case err := <-errChan:
						lastErr = err
						utils.PrintRed(fmt.Sprintf("Error processing %s: %v (attempt %d/%d)",
							path, err, attempt+1, maxRetries))
					case <-ctx.Done():
						lastErr = fmt.Errorf("timeout after %v", timeout)
						utils.PrintRed(fmt.Sprintf("Timeout processing %s after %v (attempt %d/%d)",
							path, timeout, attempt+1, maxRetries))
					}
					// Always cancel the context created with timeout
					cancel()
				}

				if !success {
					utils.PrintRed(fmt.Sprintf("Failed to process %s after %d attempts: %v",
						path, maxRetries, lastErr))
				}
			}
		}(i)
	}
	// Wait for all workers to finish
	wg.Wait()
}

// calculateOptimalWorkerCount determines the optimal number of workers based on system resources
func calculateOptimalWorkerCount() int {
	cpuCount := runtime.NumCPU()
	// Use a reasonable number of workers based on CPU count
	// Typically for I/O-bound tasks (like API calls), we can use more workers than CPUs
	// For CPU-bound tasks, we need stay close to the CPU count
	return int(
		// Between 1 and 20, with CPU*2 as guidance
		math.Max(1, math.Min(float64(cpuCount*2), 20)),
	)
}

// processTask handles a single task, separated to simplify the worker logic
func processTask(path string, secretsMap map[string]any, debug bool) error {
	// Parse the configuration for the file
	config, err := yamlparser.ParseConfig(path, secretsMap)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Handle different kinds based on the yaml 'kind' we get
	switch {
	case config.IsAuth():
		return features.SendAPIRequestForAuth2(secretsMap, path, debug)
	case config.IsAPI():
		return apicalls.SendAndSaveAPIRequest(secretsMap, path, debug)
	default:
		return fmt.Errorf("unsupported kind in file: %s", path)
	}
}
