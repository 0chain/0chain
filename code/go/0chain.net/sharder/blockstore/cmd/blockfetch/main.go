// blockfetch fetches missing blocks from external sharders via HTTP API.
// Reads a list of hashes, queries each sharder, saves raw block data as loose files.
//
// Usage:
//   blockfetch -hashes /tmp/skipped_hashes.txt -out /tmp/fetched_blocks -workers 16
package main

import (
	"bufio"
	"compress/zlib"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	extension = "dat.zlib"
	subDirs   = 5
)

var sharders = []string{
	"http://zcn-sharder.safestor.net:7171",
	"http://s01.0chainstaking.net:7171",
	"http://sharder-nl.zcn-network.eu:7171",
	"http://shard.zus-network.com:7171",
	"http://sharder1.bytepatch.io:7171",
	"http://s.sdredfox.com:7171",
}

type blockResponse struct {
	Block json.RawMessage `json:"block"`
}

func main() {
	hashFile := flag.String("hashes", "", "file with one hash per line (required)")
	outDir := flag.String("out", "", "output directory for fetched blocks (required)")
	workers := flag.Int("workers", 16, "number of parallel fetch workers")
	timeout := flag.Int("timeout", 10, "HTTP timeout in seconds per request")
	flag.Parse()

	if *hashFile == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "error: -hashes and -out required")
		flag.Usage()
		os.Exit(1)
	}
	os.MkdirAll(*outDir, 0700)

	hashes, err := readHashes(*hashFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d hashes, %d sharders, %d workers\n", len(hashes), len(sharders), *workers)

	client := &http.Client{Timeout: time.Duration(*timeout) * time.Second}

	hashCh := make(chan string, 1000)
	var wg sync.WaitGroup
	var fetched, failed, notfound int64
	start := time.Now()

	// Stats printer
	go func() {
		for {
			time.Sleep(60 * time.Second)
			f := atomic.LoadInt64(&fetched)
			nf := atomic.LoadInt64(&notfound)
			fl := atomic.LoadInt64(&failed)
			elapsed := time.Since(start).Round(time.Second)
			rate := float64(f+nf+fl) / elapsed.Seconds()
			fmt.Fprintf(os.Stderr, "  fetched=%d notfound=%d failed=%d rate=%.0f/s elapsed=%v\n", f, nf, fl, rate, elapsed)
		}
	}()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for hash := range hashCh {
				if fetchBlock(client, hash, *outDir) {
					atomic.AddInt64(&fetched, 1)
				} else {
					atomic.AddInt64(&notfound, 1)
				}
			}
		}()
	}

	for _, h := range hashes {
		hashCh <- h
	}
	close(hashCh)
	wg.Wait()

	elapsed := time.Since(start).Round(time.Second)
	fmt.Fprintf(os.Stderr, "Done. fetched=%d notfound=%d failed=%d in %v\n",
		atomic.LoadInt64(&fetched), atomic.LoadInt64(&notfound), atomic.LoadInt64(&failed), elapsed)
}

func fetchBlock(client *http.Client, hash, outDir string) bool {
	for _, sharder := range sharders {
		url := fmt.Sprintf("%s/v1/block/get?block=%s&content=full", sharder, hash)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || len(body) == 0 {
			continue
		}

		// Save as zlib-compressed loose file
		if err := saveBlock(hash, body, outDir); err != nil {
			continue
		}
		return true
	}
	return false
}

func saveBlock(hash string, data []byte, outDir string) error {
	if len(hash) < subDirs {
		return fmt.Errorf("invalid hash")
	}
	var s string
	for i := 0; i < subDirs; i++ {
		s += string(hash[i]) + string(os.PathSeparator)
	}
	path := filepath.Join(outDir, s, hash[subDirs:]+"."+extension)
	os.MkdirAll(filepath.Dir(path), 0700)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := zlib.NewWriterLevel(f, zlib.BestCompression)
	if err != nil {
		return err
	}
	w.Write(data)
	return w.Close()
}

func readHashes(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var hashes []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		h := scanner.Text()
		if len(h) == 64 {
			hashes = append(hashes, h)
		}
	}
	return hashes, scanner.Err()
}
