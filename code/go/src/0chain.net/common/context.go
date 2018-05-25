package common

import (
	"context"
	"fmt"
)

var rootContext context.Context
var rootCancel context.CancelFunc
var done chan bool

/*SetupRootContext - sets up the root context that can be used to shutdown the node */
func SetupRootContext(nodectx context.Context) {
	done = make(chan bool)
	rootContext, rootCancel = context.WithCancel(nodectx)
	// TODO: This go routine is not needed. Workaround for the "vet" error
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
	if rootContext == nil { // TODO: This shouldn't be there but some package initializion is using GetRootContext before the SetupRootContext is created
		SetupRootContext(context.Background())
	}
	return rootContext
}

/*Done - call this when the program needs to stop and notify all workers */
func Done() {
	fmt.Printf("Initiating shutdown...\n")
	rootCancel()
	// done <- true
}
