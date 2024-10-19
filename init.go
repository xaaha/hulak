package main

import (
	"github.com/xaaha/hulak/pkg/envparser"
)

/*
InitializeProject() starts the project by creating envfolder and global file in it.
TBC...
*/
func InitializeProject() {
	err := envparser.CreateDefaultEnvs(nil)
	if err != nil {
		panic(err)
	}
}
