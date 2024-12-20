package main

import (
	"flag"

	userflags "github.com/xaaha/hulak/pkg/userFlags"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	// parse all the necessary flags
	flag.Parse()

	env := userflags.Env()
	fp := userflags.FilePath()
	fileName := userflags.File()

	// create envMap
	envMap := InitializeProject(env)

	filePathList, err := userflags.GenerateFilePathList(fileName, fp)
	if err != nil {
		utils.PanicRedAndExit("main.go %v", err)
	}

	RunTasks(filePathList, envMap)
}

/*
- How do we handle making a file dependent on a response of different request call?
  - Save the file to it's respective file type concurrently
  - Once the file is done saving, lock it and copy the content in a buffer (mutex)
  - Then release the content
  - Then use the content to recursively find the key mentioned in the template {{getValueOf accesstoken employee_auth}}
    - This is where I need to make the change pkg/envparser/replaceVars.go
    - function name, value you want to grab, and file name you want to grab from.
    - File name should be unique. Or Provide file path. If there are multiple paths found, use the first one
    -
  - Also, need to think about how to handle a situation where the call you are depending on is dependent on another call and they are all concurrently running
     - Do we not use the concurrency at that time?
     - For example, {{getValueOf user_uuid employeeDetails}}
     - But, employeeDetails file has {{getValueOf foo bar}} and they are all running concurrently. Do we run one after the other in that case?
     - since we are calling all the files concurrently, we need to lock a json file with mutex, and then check if other file is making the call.
      Last thing we need is to multiple calls calling the same AuthCall
     - Use channel to communicate the response before writing the file?
*/
