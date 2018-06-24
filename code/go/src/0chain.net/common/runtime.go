package common

import (
	"runtime"

	"go.uber.org/zap"
)

const MB = 1024 * 1024

/*LogRuntime - log the current runtime statistics to the given log */
func LogRuntime(logger *zap.Logger, ref zap.Field) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	logger.Info("runtime", ref, zap.Int("goroutines", runtime.NumGoroutine()), zap.Uint64("heap_objects", mem.HeapObjects), zap.Uint32("gc", mem.NumGC), zap.Uint64("gc_pause", mem.PauseNs[(mem.NumGC+255)%256]))
	logger.Info("runtime", ref, zap.Uint64("total_alloc", mem.TotalAlloc/MB), zap.Uint64("sys", mem.Sys/MB), zap.Uint64("heap_sys", mem.HeapSys/MB), zap.Uint64("heap_alloc", mem.HeapAlloc/MB))

}
