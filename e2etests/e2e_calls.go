package e2etests

import (
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
)

// takes in the environment secrets map and file path (fp) from user flag
func RunFormData(envMap map[string]string, fp string) {
	apicalls.SendApiRequest(envMap, fp)
}

func RunFormDataError(envMap map[string]string) {
	apicalls.SendApiRequest(envMap, "e2etests/test_collection/form_data_error.yml")
}

func RunUrlEncodedFormData(envMap map[string]string) {
	apicalls.SendApiRequest(envMap, "e2etests/test_collection/url_encoded_form.yaml")
}

/*
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
*/
