package main

import (
	"fmt"
	"os"

	"0chain.net/smartcontract/benchmark/main/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("error: %w", err)
		}
	}()
}
