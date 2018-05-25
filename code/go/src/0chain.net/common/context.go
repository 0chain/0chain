package common

import (
	"context"
	"fmt"
)

var rootContext context.Context
var rootCancel context.CancelFunc
var done chan bool

func init() {
	done = make(chan bool)
	rootContext, rootCancel = context.WithCancel(context.Background())
	go func() {
		select {
		case <-done:
			fmt.Printf("Shutting down all workers...\n")
			rootCancel()
		}
	}()
}

/*GetRootContext - get the root context for the server
* This will be used to control shutting down the server but cleanup all the workers
 */
func GetRootContext() context.Context {
	return rootContext
}

/*Done - call this when the program needs to stop and notify all workers */
func Done() {
	fmt.Printf("Initiating shutdown...\n")
	done <- true
}
