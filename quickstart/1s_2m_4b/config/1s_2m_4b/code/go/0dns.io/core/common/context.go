package common

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var rootContext context.Context
var rootCancel context.CancelFunc

/*SetupRootContext - sets up the root context that can be used to shutdown the node */
func SetupRootContext(nodectx context.Context) {
	rootContext, rootCancel = context.WithCancel(nodectx)
	// TODO: This go routine is not needed. Workaround for the "vet" error
	done := make(chan bool)
	go func() {
		select {
		case <-done:
			//Logger.Info("Shutting down all workers...")
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
	//Logger.Info("Initiating shutdown...")
	rootCancel()
	//TODO: How do we ensure every worker is completed any shutdown sequence before we finally shut down
	//the server using server.Shutdown(ctx)
}

/*HandleShutdown - handles various shutdown signals */
func HandleShutdown(server *http.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGINT:
				Done()
				ctx, cancelf := context.WithTimeout(context.Background(), 5*time.Second)
				server.Shutdown(ctx)
				cancelf()
			case syscall.SIGQUIT:
				Done()
				ctx, cancelf := context.WithTimeout(context.Background(), 5*time.Second)
				server.Shutdown(ctx)
				cancelf()
			default:
				//Logger.Debug("unhandled signal", zap.Any("signal", sig))
			}
		}
	}()
}
