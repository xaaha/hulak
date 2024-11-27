package main

import (
	"sync"

	"github.com/xaaha/hulak/e2etests"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	envMap := InitializeProject()
	// apicalls.TestApiCalls() // temp call.. replace with mock

	var wg sync.WaitGroup
	wg.Add(3) // number of go routines

	go func() {
		defer wg.Done()
		e2etests.RunFormData(utils.CopyEnvMap(envMap))
	}()
	go func() {
		defer wg.Done()
		e2etests.RunUrlEncodedFormData(utils.CopyEnvMap(envMap))
	}()
	go func() {
		defer wg.Done()
		e2etests.RunFormDataError(utils.CopyEnvMap(envMap))
	}()
	wg.Wait()
}
