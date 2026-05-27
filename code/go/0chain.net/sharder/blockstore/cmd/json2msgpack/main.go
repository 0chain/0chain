// json2msgpack converts block files from JSON+zlib to msgpack+zlib format.
// Walks a directory of .dat.zlib files, reads each as zlib(JSON),
// decodes to block.Block struct, re-encodes as zlib(msgpack).
//
// Must be built with the 0chain block package (requires CGO for rocksdb).
// Build inside Docker: use the sharder Dockerfile build stage.
//
// Usage: json2msgpack -dir /tmp/fetched_blocks -out /tmp/converted_blocks -workers 8
package main

import (
	"compress/zlib"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
)

func main() {
	inDir := flag.String("dir", "", "input directory with JSON+zlib .dat.zlib files (required)")
	outDir := flag.String("out", "", "output directory for msgpack+zlib files (default: overwrite in place)")
	workers := flag.Int("workers", 8, "parallel workers")
	flag.Parse()

	if *inDir == "" {
		fmt.Fprintln(os.Stderr, "error: -dir required")
		os.Exit(1)
	}

	if *outDir == "" {
		*outDir = *inDir
	}

	var files []string
	filepath.Walk(*inDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".dat.zlib") {
			files = append(files, path)
		}
		return nil
	})

	fmt.Fprintf(os.Stderr, "Found %d files\n", len(files))

	ch := make(chan string, 1000)
	var converted, failed int64
	var wg sync.WaitGroup
	start := time.Now()

	go func() {
		for {
			time.Sleep(30 * time.Second)
			fmt.Fprintf(os.Stderr, "  converted=%d failed=%d elapsed=%v\n",
				atomic.LoadInt64(&converted), atomic.LoadInt64(&failed),
				time.Since(start).Round(time.Second))
		}
	}()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range ch {
				rel, _ := filepath.Rel(*inDir, path)
				outPath := filepath.Join(*outDir, rel)
				if convertFile(path, outPath) {
					atomic.AddInt64(&converted, 1)
				} else {
					atomic.AddInt64(&failed, 1)
				}
			}
		}()
	}

	for _, f := range files {
		ch <- f
	}
	close(ch)
	wg.Wait()

	fmt.Fprintf(os.Stderr, "Done. converted=%d failed=%d in %v\n",
		atomic.LoadInt64(&converted), atomic.LoadInt64(&failed),
		time.Since(start).Round(time.Second))
}

var debugOnce sync.Once

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func convertFile(inPath, outPath string) bool {
	f, err := os.Open(inPath)
	if err != nil {
		return false
	}

	r, err := zlib.NewReader(f)
	if err != nil {
		f.Close()
		return false
	}

	jsonData, err := io.ReadAll(r)
	r.Close()
	f.Close()
	if err != nil {
		return false
	}

	// Unwrap {"block": {...}} if present
	var wrapper struct {
		Block json.RawMessage `json:"block"`
	}
	blockJSON := jsonData
	if err := json.Unmarshal(jsonData, &wrapper); err == nil && len(wrapper.Block) > 0 {
		blockJSON = wrapper.Block
	}

	// Decode JSON to block.Block
	b := &block.Block{}
	if err := json.Unmarshal(blockJSON, b); err != nil {
		debugOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "DEBUG unmarshal error on %s: %v\nJSON prefix: %s\n", inPath, err, string(blockJSON[:minInt(200, len(blockJSON))]))
		})
		return false
	}

	// Write msgpack+zlib
	os.MkdirAll(filepath.Dir(outPath), 0700)
	out, err := os.Create(outPath + ".tmp")
	if err != nil {
		return false
	}

	w, err := zlib.NewWriterLevel(out, zlib.BestCompression)
	if err != nil {
		out.Close()
		os.Remove(outPath + ".tmp")
		return false
	}

	if err := datastore.WriteMsgpack(w, b); err != nil {
		w.Close()
		out.Close()
		os.Remove(outPath + ".tmp")
		return false
	}

	w.Close()
	out.Close()
	os.Rename(outPath+".tmp", outPath)
	return true
}
