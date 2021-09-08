package main

import (
	"context"

	"0chain.net/smartcontract/magmasc/benchmark/state-generator/cli"
)

func main() {
	app := cli.New()

	if err := cli.Start(context.Background(), app); err != nil {
		panic(err)
	}
}
