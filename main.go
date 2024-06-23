package main

import (
	"log"
	"os"

	"github.com/juicymango/yeah_woo_go/handler"
)

func main() {
	// Check to make sure the path is provided as an argument.
	if len(os.Args) < 2 {
		log.Fatal("Please provide a file path as an argument.")
	}

	// The first argument is always the program name, so the second argument is the file path.
	filePath := os.Args[1]

	handler.Handle(filePath)
}
