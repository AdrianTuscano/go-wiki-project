package main

import (
	"fmt"
	// "go-wiki-project/mypackage"
	"go-wiki-project/wiki"
)

func main() {
	fmt.Println("Hello, World!")
	// mypackage.PrintHello()

	wiki.Init()
	wiki.Main()

}
