package utils

func InitializeProject() {
	/*
	  - Make sure .env folder exists in the root of the project
	  - navigate to each folder in the root  to look for the .env folder.
	    - root is .
	    - get cwd root by default

	  ```bash
	  $ hulak "dir/path or ."
	  ```
	  how do you persist
	  - if the folder does not exist, create a folder and a file and exit
	  - if the folder exists, but the file does not exsts, then create the file
	  and exit.
	  - Otherwise, look for the global.env by default. Otherwise, look for the the
	  environment specified by the user
	  - Something like what lazy git does with *
	  - Always look for active environment, collection, and then global
	  - if the variable is not found then return the error in red
	*/
}
