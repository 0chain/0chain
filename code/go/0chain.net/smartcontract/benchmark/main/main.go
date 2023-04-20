package main

import (
	"fmt"

	"0chain.net/smartcontract/benchmark/main/cmd"
)

func main() {
	_ = cmd.Execute()

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("error: %w", err)
		}
	}()
}
