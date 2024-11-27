package main

import (
	"sync"

	"github.com/xaaha/hulak/e2etests"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	envMap := InitializeProject()
	var wg sync.WaitGroup
	// apicalls.TestApiCalls() // temp call.. replace with mock

	tasks := []func(map[string]string){
		e2etests.RunFormData,
		e2etests.RunUrlEncodedFormData,
		e2etests.RunFormDataError,
	}

	for _, task := range tasks {
		wg.Add(1) // one for each task
		go func(taskfunc func(map[string]string), env map[string]string) {
			defer wg.Done()
			taskfunc(utils.CopyEnvMap(env))
		}(task, envMap)
	}
	wg.Wait()
}
