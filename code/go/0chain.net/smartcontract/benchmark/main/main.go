package main

import (
	"0chain.net/smartcontract/benchmark/main/cmd"
	"fmt"
)

func main() {
	_ = cmd.Execute()

	defer func() {
		if err := recover(); err != nil {
			_ = fmt.Errorf("error: %v", err)
		}
	}()
}
