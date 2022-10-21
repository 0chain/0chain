package log

import (
	"log"
	"runtime/debug"

	bk "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

var (
	verbose = true
)

func Println(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func Fatal(v ...interface{}) {
	debug.PrintStack()
	log.Println("fatal debug")
	log.Fatal(v...)
}

func SetVerbose(v bool) {
	verbose = v
}

func GetVerbose() bool {
	return verbose
}

func PrintSimSettings() {
	if verbose {
		for i := bk.SimulatorParameter(0); i < bk.NumberSimulationParameters; i++ {
			println(i.String(), viper.GetInt(bk.Simulation+i.String()))
		}
		println()
	}
}
