// Package runner contains the API request execution pipeline.
// It is imported by both the run subcommand and main's interactive mode.
package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"time"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/features"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// Flags holds parsed CLI flags needed by the execution pipeline.
type Flags struct {
	Env      string
	EnvSet   bool
	FilePath string
	File     string
	Debug    bool
	Dir      string
	Dirseq   string
}

// Execute runs the full pipeline: discover files, resolve env, execute requests.
func Execute(f *Flags) {
	fileList, concurrentDir, sequentialDir := discoverFilePaths(
		f.File,
		f.FilePath,
		f.Dir,
		f.Dirseq,
		f.Dir != "" || f.Dirseq != "",
	)

	allPaths := append(append(fileList, concurrentDir...), sequentialDir...)

	var envMap map[string]any
	if containsTemplateVars(allPaths) {
		if !utils.IsHulakProject() {
			utils.PanicRedAndExit("fatal: not a hulak project \n\nRun 'hulak init' to set up")
		}
		envMap = InitializeProject(f.Env, true)
	}

	handleAPIRequests(
		envMap,
		f.Debug,
		append(fileList, concurrentDir...),
		sequentialDir,
		f.FilePath,
	)
}

// ExecuteSingleFile runs a single file through the pipeline.
// Used by interactive mode where the file is already known.
func ExecuteSingleFile(envMap map[string]any, debug bool, filePath string) {
	handleAPIRequests(envMap, debug, []string{filePath}, nil, filePath)
}

// containsTemplateVars returns true if any file in the list uses template vars.
func containsTemplateVars(paths []string) bool {
	return slices.ContainsFunc(paths, utils.FileHasTemplateVars)
}

// InitializeProject creates the env setup and returns the secrets map.
func InitializeProject(env string, isCli bool) map[string]any {
	if err := envparser.CreateDefaultEnvs(nil); err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	envMap, err := envparser.GenerateSecretsMap(env, isCli)
	if err != nil {
		panic(err)
	}
	return envMap
}

// discoverFilePaths collects all file paths from -f, -fp, -dir, and -dirseq flags.
func discoverFilePaths(
	fileName, fp, dir, dirseq string,
	hasDirFlags bool,
) (fileList, concurrentDir, sequentialDir []string) {
	if fp != "" || fileName != "" {
		var err error
		fileList, err = generateFilePathList(fileName, fp)
		if err != nil {
			if !hasDirFlags {
				utils.PanicRedAndExit("%v", err)
			}
			utils.PrintWarning(fmt.Sprintf("Warning with file flags: %v", err))
		}
	}

	if hasDirFlags {
		dirPaths, err := apicalls.ListDirPaths(dir, dirseq)
		if err != nil {
			utils.PrintRed(fmt.Sprintf("Error processing directories: %v", err))
		} else {
			concurrentDir = dirPaths.Concurrent
			sequentialDir = dirPaths.Sequential
		}
	}

	return fileList, concurrentDir, sequentialDir
}

// handleAPIRequests processes API requests from pre-discovered file lists.
func handleAPIRequests(
	secrets map[string]any,
	debug bool,
	concurrentFiles []string,
	sequentialFiles []string,
	fp string,
) {
	if len(concurrentFiles) > 0 {
		if len(concurrentFiles) > 1 || len(sequentialFiles) > 0 {
			utils.PrintInfo(
				fmt.Sprintf("Processing %d files concurrently...", len(concurrentFiles)),
			)
		}
		runTasks(concurrentFiles, secrets, debug, fp)
	}
	if len(sequentialFiles) > 0 {
		utils.PrintInfo(fmt.Sprintf("Processing %d files sequentially...", len(sequentialFiles)))
		processFilesSequentially(sequentialFiles, secrets, debug)
	}

	totalFiles := len(concurrentFiles) + len(sequentialFiles)
	if totalFiles == 0 {
		utils.PrintWarning(
			"No files were processed. Please check your path or directory arguments.",
		)
	}
}

// runTasks manages the go tasks with a limited worker pool
func runTasks(filePathList []string, secretsMap map[string]any, debug bool, fp string) {
	maxWorkers := utils.GetWorkers(nil)
	maxRetries := 3
	timeout := 60 * time.Second

	if fp != "" {
		maxRetries = 1
	}

	var wg sync.WaitGroup
	taskChan := make(chan string, len(filePathList))

	for _, path := range filePathList {
		taskChan <- path
	}
	close(taskChan)

	for i := range maxWorkers {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			for path := range taskChan {
				success := false
				var lastErr error

				for attempt := 0; attempt < maxRetries && !success; attempt++ {
					if attempt > 0 {
						backoffDuration := time.Duration(1<<(attempt-1)) * time.Second
						utils.PrintWarning(fmt.Sprintf("Retrying %s (attempt %d/%d) after %v",
							path, attempt+1, maxRetries, backoffDuration))
						time.Sleep(backoffDuration)
					}

					ctx, cancel := context.WithTimeout(context.Background(), timeout)

					doneChan := make(chan struct{})
					errChan := make(chan error, 1)

					go func() {
						err := processTask(path, utils.CopyEnvMap(secretsMap), debug)
						if err != nil {
							errChan <- err
						} else {
							close(doneChan)
						}
					}()

					select {
					case <-doneChan:
						success = true
						utils.PrintInfo(fmt.Sprintf("Processed '%s'", filepath.Base(path)))
					case err := <-errChan:
						lastErr = err
						utils.PrintInfo(fmt.Sprintf("(attempt %d/%d)", attempt+1, maxRetries))
					case <-ctx.Done():
						lastErr = fmt.Errorf("timeout after %v", timeout)
						utils.PrintRed(fmt.Sprintf("Timeout processing %s after %v (attempt %d/%d)",
							path, timeout, attempt+1, maxRetries))
					}
					cancel()
				}

				if !success {
					utils.PrintRed(fmt.Sprintf("Failed to process %s after %d attempts: %v",
						path, maxRetries, lastErr))
				}
			}
		}(i)
	}
	wg.Wait()
}

// processTask handles a single task
func processTask(path string, secretsMap map[string]any, debug bool) error {
	config, err := yamlparser.ParseConfig(path, secretsMap)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse config %v", err)
		return utils.ColorError(errMsg)
	}

	switch {
	case config.IsAuth():
		return features.SendAPIRequestForAuth2(secretsMap, path, debug)
	case (config.IsAPI() || config.IsGraphql()):
		return apicalls.SendAndSaveAPIRequest(secretsMap, path, debug)
	default:
		return fmt.Errorf("unsupported kind in file: %s", path)
	}
}

// processFilesSequentially handles files one by one
func processFilesSequentially(filePaths []string, secretsMap map[string]any, debug bool) {
	for _, path := range filePaths {
		fileEnv := utils.CopyEnvMap(secretsMap)

		err := processTask(path, fileEnv, debug)
		utils.PrintInfo(fmt.Sprintf("Processed: '%s'", filepath.Base(path)))
		if err != nil {
			utils.PrintRed(fmt.Sprintf("Error processing %s: %v", path, err))
		}
	}
}

// generateFilePathList returns a slice of file paths based on the flags -f and -fp.
func generateFilePathList(fileName string, fp string) ([]string, error) {
	standardErrMsg := "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' "

	if fileName == "" && fp == "" {
		return nil, utils.ColorError(standardErrMsg)
	}

	var filePathList []string

	if fp != "" {
		filePathList = append(filePathList, fp)
	}

	if fileName != "" {
		if matchingPaths, err := utils.ListMatchingFiles(fileName); err != nil {
			utils.PrintRed(utils.ErrFilePathCollection + ": " + err.Error())
		} else {
			filePathList = append(filePathList, matchingPaths...)
		}
	}

	if len(filePathList) == 0 {
		return nil, utils.ColorError(standardErrMsg)
	}
	return filePathList, nil
}
