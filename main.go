package main

import "github.com/xaaha/hulak/e2etests"

func main() {
	envMap := InitializeProject()
	// apicalls.TestApiCalls() // temp call.. replace with mock
	e2etests.RunFormData(envMap)
	e2etests.RunUrlEncodedFormData(envMap)
	e2etests.RunFormDataError(envMap)
}
