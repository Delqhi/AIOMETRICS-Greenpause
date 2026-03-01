package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("usage: cli <command>")
		fmt.Println("commands: health")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "health":
		fmt.Println("ok")
	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		os.Exit(2)
	}
}
