// buildindex creates global.idx from all pack files in a packs directory.
// Run after compaction to enable O(log N) hash→pack lookup.
//
// Usage: buildindex -packs /path/to/blocks/packs
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	packMagic      = "BPAK"
	hashLen        = 64
	indexEntrySize = hashLen + 8 + 4
	trailerSize    = 16

	globalIdxMagic    = "GIDX"
	globalEntrySize   = hashLen + 2
	globalTrailerSize = 12
)

type gEntry struct {
	hash   [hashLen]byte
	packID uint16
}

func main() {
	packsDir := flag.String("packs", "", "path to packs directory (required)")
	flag.Parse()

	if *packsDir == "" {
		fmt.Fprintln(os.Stderr, "error: -packs flag is required")
		flag.Usage()
		os.Exit(1)
	}

	start := time.Now()
	fmt.Println("Scanning pack files...")

	entries, err := os.ReadDir(*packsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var packNames []string
	var allEntries []gEntry
	totalBlocks := 0

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pack") {
			continue
		}
		path := filepath.Join(*packsDir, e.Name())
		hashes, err := readPackHashes(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", e.Name(), err)
			continue
		}

		packID := uint16(len(packNames))
		packNames = append(packNames, e.Name())

		for _, h := range hashes {
			var entry gEntry
			copy(entry.hash[:], h)
			entry.packID = packID
			allEntries = append(allEntries, entry)
		}
		totalBlocks += len(hashes)

		if len(packNames)%1000 == 0 {
			fmt.Printf("  %d packs scanned (%d blocks)...\n", len(packNames), totalBlocks)
		}
	}

	fmt.Printf("Scanned %d packs, %d blocks in %v\n", len(packNames), totalBlocks, time.Since(start).Round(time.Second))
	fmt.Println("Sorting by hash...")

	sort.Slice(allEntries, func(i, j int) bool {
		return bytes.Compare(allEntries[i].hash[:], allEntries[j].hash[:]) < 0
	})

	fmt.Println("Writing global.idx...")

	outPath := filepath.Join(*packsDir, "global.idx")
	f, err := os.Create(outPath + ".tmp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating file: %v\n", err)
		os.Exit(1)
	}

	// Header: packCount + pack names
	binary.Write(f, binary.LittleEndian, uint16(len(packNames)))
	for _, name := range packNames {
		binary.Write(f, binary.LittleEndian, uint16(len(name)))
		f.Write([]byte(name))
	}
	headerEnd, _ := f.Seek(0, 1)

	// Entries: sorted {hash[64] + packID[2]}
	for _, e := range allEntries {
		var buf [globalEntrySize]byte
		copy(buf[:hashLen], e.hash[:])
		binary.LittleEndian.PutUint16(buf[hashLen:], e.packID)
		f.Write(buf[:])
	}

	// Trailer
	var trailer [globalTrailerSize]byte
	binary.LittleEndian.PutUint32(trailer[:4], uint32(len(allEntries)))
	binary.LittleEndian.PutUint32(trailer[4:8], uint32(headerEnd))
	copy(trailer[8:], globalIdxMagic)
	f.Write(trailer[:])

	f.Sync()
	f.Close()
	os.Rename(outPath+".tmp", outPath)

	fi, _ := os.Stat(outPath)
	fmt.Printf("Done. global.idx: %dMB (%d entries, %d packs) in %v\n",
		fi.Size()/(1024*1024), len(allEntries), len(packNames), time.Since(start).Round(time.Second))
}

func readPackHashes(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() < int64(trailerSize) {
		return nil, fmt.Errorf("too small")
	}

	var trailer [trailerSize]byte
	f.ReadAt(trailer[:], fi.Size()-int64(trailerSize))
	if string(trailer[12:16]) != packMagic {
		return nil, fmt.Errorf("bad magic")
	}

	indexOffset := binary.LittleEndian.Uint64(trailer[:8])
	indexCount := binary.LittleEndian.Uint32(trailer[8:12])

	indexBuf := make([]byte, int64(indexCount)*int64(indexEntrySize))
	if _, err := f.ReadAt(indexBuf, int64(indexOffset)); err != nil {
		return nil, err
	}

	hashes := make([]string, indexCount)
	for i := uint32(0); i < indexCount; i++ {
		off := int64(i) * int64(indexEntrySize)
		hashes[i] = string(bytes.TrimRight(indexBuf[off:off+hashLen], "\x00"))
	}
	return hashes, nil
}
