package log

import "log"

var (
	verbose = true
)

func Println(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

func SetVerbose(v bool) {
	verbose = v
}
