package graphql

import "fmt"

func Introspect(args []string) {
	fmt.Println("Hello World!")
	// TODO: Handle this
	for idx, val := range args {
		fmt.Println(idx+1, val)
	}
}
