// Package main initializes the project and runs the query
package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
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

// runTasks manages the go tasks with a limited worker pool
func runTasks(filePathList []string, secretsMap map[string]any, debug bool) {
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

/*
 things we could optiize for but is probably too much here
 Rate limiting an API.
 Priority Queues: For mixed task types with different priorities
 Result Collection: Add a results channel to collect success/failure statistics.
 Graceful Shutdown: Add signal handling to cancel in-progress tasks if the program is terminated.
*/

// HandleAPIRequests processes API requests, and runs taks from individual files and directories
// It also includes Auth2.0 call
// Handling both concurrent (-dir) and sequential (-dirseq) processing
func HandleAPIRequests(
	secretsMap map[string]any,
	debug bool,
	filePathList []string,
	dir, dirseq string,
) {
	var allFiles []string
	var sequentialFiles []string

	// Add existing file paths to the concurrent processing list
	if len(filePathList) > 0 {
		allFiles = append(allFiles, filePathList...)
	}

	// Process directory paths if provided
	if dir != "" || dirseq != "" {
		dirPaths, err := apicalls.ListDirPaths(dir, dirseq)

		if err != nil {
			utils.PrintRed(fmt.Sprintf("Error processing directories: %v", err))
		} else {
			// Add concurrent directory files to the main processing list
			if len(dirPaths.Concurrent) > 0 {
				allFiles = append(allFiles, dirPaths.Concurrent...)
			}

			// Keep sequential files separate
			if len(dirPaths.Sequential) > 0 {
				sequentialFiles = append(sequentialFiles, dirPaths.Sequential...)
			}
		}
	}

	// Process concurrent files if any
	if len(allFiles) > 0 {
		if dir != "" || dirseq != "" {
			utils.PrintInfo(fmt.Sprintf("Processing %d files concurrently...", len(allFiles)))
		}
		runTasks(allFiles, secretsMap, debug)
	}

	// Process sequential files one by one
	if len(sequentialFiles) > 0 {
		utils.PrintInfo(fmt.Sprintf("Processing %d files sequentially...", len(sequentialFiles)))
		processFilesSequentially(sequentialFiles, secretsMap, debug)
	}

	totalFiles := len(allFiles) + len(sequentialFiles)
	if totalFiles < 0 {
		utils.PrintWarning(
			"No files were processed. Please check your path or directory arguments.",
		)
	}
}

// processFilesSequentially handles files one by one in a sequential manner
func processFilesSequentially(filePaths []string, secretsMap map[string]any, debug bool) {
	for _, path := range filePaths {
		utils.PrintInfo(filepath.Base(path))

		// Create a fresh copy of the environment for each file
		fileEnv := utils.CopyEnvMap(secretsMap)

		err := processTask(path, fileEnv, debug)
		if err != nil {
			utils.PrintRed(fmt.Sprintf("Error processing %s: %v", path, err))
		} else if debug {
			utils.PrintInfo(fmt.Sprintf("Successfully processed %s", path))
		}
	}
}
