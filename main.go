package main

import (
	"flag"
	"sync"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	userflags "github.com/xaaha/hulak/pkg/userFlags"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	// parse all the necessary flags
	flag.Parse()

	fp := userflags.FilePath()
	env := userflags.Env()

	// create envMap
	envMap := InitializeProject(env)

	var wg sync.WaitGroup

	// Define tasks
	tasks := []func(map[string]string, string){
		apicalls.SendApiRequest,
	}

	// Run tasks concurrently
	for _, task := range tasks {
		wg.Add(1)
		go func(taskfunc func(map[string]string, string), env map[string]string, filePath string) {
			defer wg.Done()
			taskfunc(utils.CopyEnvMap(env), filePath)
		}(task, envMap, fp)
	}

	wg.Wait()
}
