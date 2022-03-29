package common

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"go.uber.org/zap"

	"0chain.net/core/viper"
)

const MB = 1024 * 1024

/*LogRuntime - log the current runtime statistics to the given log */
func LogRuntime(logger *zap.Logger, ref zap.Field) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	logger.Info("runtime", ref, zap.Int("goroutines", runtime.NumGoroutine()), zap.Uint64("heap_objects", mem.HeapObjects), zap.Uint32("gc", mem.NumGC), zap.Uint64("gc_pause", mem.PauseNs[(mem.NumGC+255)%256]))
	logger.Info("runtime", ref, zap.Uint64("total_alloc", mem.TotalAlloc/MB), zap.Uint64("sys", mem.Sys/MB), zap.Uint64("heap_sys", mem.HeapSys/MB), zap.Uint64("heap_alloc", mem.HeapAlloc/MB))
	if viper.GetBool("logging.goroutines") {
		_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	}
}

// WaitSigInt blocks until SIGIN received. It logs about it to STDOUT.
func WaitSigInt() {
	var c = make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // the Kill doesn't matter
	fmt.Printf("got signal %s, exiting...\n", <-c)
}
