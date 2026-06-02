// targetpack packs specific blocks from loose files into pack files.
// Reads a list of hashes (one per line) from a file, reads each loose block
// from the block tree, and writes them into 10K-block packs.
//
// Usage:
//   targetpack -blocks /path/to/blocks -hashes /tmp/gap_hashes.txt -batch 10000
//
// The hashes file should contain one 64-char hex hash per line.
// Blocks that don't exist as loose files are skipped (counted).
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	packMagic      = "BPAK"
	hashLen        = 64
	indexEntrySize = hashLen + 8 + 4
	trailerSize    = 16
	extension      = "dat.zlib"
	subDirs        = 5
)

type indexEntry struct {
	hash   string
	offset uint64
	length uint32
}

func main() {
	blocksDir := flag.String("blocks", "", "block store base path (required)")
	hashesFile := flag.String("hashes", "", "file with one hash per line (required)")
	batchSize := flag.Int("batch", 10000, "blocks per pack")
	outDir := flag.String("out", "", "output directory for packs (default: <blocks>/packs)")
	flag.Parse()

	if *blocksDir == "" || *hashesFile == "" {
		fmt.Fprintln(os.Stderr, "error: -blocks and -hashes flags required")
		flag.Usage()
		os.Exit(1)
	}

	if *outDir == "" {
		*outDir = filepath.Join(*blocksDir, "packs")
	}
	os.MkdirAll(*outDir, 0700)

	// Read hashes
	hashes, err := readHashList(*hashesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading hashes: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d target hashes\n", len(hashes))

	start := time.Now()
	seq := detectNextSeq(*outDir, "target")
	var batch []blockData
	totalPacked := 0
	totalSkipped := 0

	for i, hash := range hashes {
		data, err := readLooseBlock(*blocksDir, hash)
		if err != nil {
			totalSkipped++
			continue
		}

		batch = append(batch, blockData{hash: hash, data: data})

		if len(batch) >= *batchSize {
			packPath := filepath.Join(*outDir, fmt.Sprintf("target_%04d.pack", seq))
			n := writePack(batch, packPath)
			totalPacked += n
			elapsed := time.Since(start).Round(time.Second)
			rate := 0
			if elapsed.Seconds() > 0 {
				rate = int(float64(totalPacked) / elapsed.Seconds())
			}
			fmt.Fprintf(os.Stderr, "  pack %d: %d blocks, %s, %d blk/s, %d skipped, %d/%d processed\n",
				seq, n, fmtSize(packPath), rate, totalSkipped, i+1, len(hashes))
			seq++
			batch = batch[:0]
		}
	}

	// Flush remaining
	if len(batch) > 0 {
		packPath := filepath.Join(*outDir, fmt.Sprintf("target_%04d.pack", seq))
		n := writePack(batch, packPath)
		totalPacked += n
		fmt.Fprintf(os.Stderr, "  pack %d: %d blocks (final), %s\n", seq, n, fmtSize(packPath))
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Fprintf(os.Stderr, "Done. %d packed, %d skipped, %d total in %v\n",
		totalPacked, totalSkipped, len(hashes), elapsed)
}

type blockData struct {
	hash string
	data []byte
}

func readLooseBlock(blocksDir, hash string) ([]byte, error) {
	if len(hash) < subDirs {
		return nil, fmt.Errorf("invalid hash")
	}
	var s string
	for i := 0; i < subDirs; i++ {
		s += string(hash[i]) + string(os.PathSeparator)
	}
	path := filepath.Join(blocksDir, s, hash[subDirs:]+"."+extension)
	return os.ReadFile(path)
}

func writePack(blocks []blockData, packPath string) int {
	tmpPath := packPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create error: %v\n", err)
		return 0
	}

	var entries []indexEntry
	var offset uint64

	for _, b := range blocks {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(b.data)))
		f.Write(lenBuf[:])
		f.Write(b.data)
		entries = append(entries, indexEntry{
			hash:   b.hash,
			offset: offset + 4,
			length: uint32(len(b.data)),
		})
		offset += 4 + uint64(len(b.data))
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].hash < entries[j].hash
	})

	indexOffset := offset
	for _, e := range entries {
		var buf [indexEntrySize]byte
		copy(buf[:hashLen], e.hash)
		binary.LittleEndian.PutUint64(buf[hashLen:], e.offset)
		binary.LittleEndian.PutUint32(buf[hashLen+8:], e.length)
		f.Write(buf[:])
	}

	var trailer [trailerSize]byte
	binary.LittleEndian.PutUint64(trailer[:8], indexOffset)
	binary.LittleEndian.PutUint32(trailer[8:12], uint32(len(entries)))
	copy(trailer[12:], packMagic)
	f.Write(trailer[:])

	f.Sync()
	f.Close()
	os.Rename(tmpPath, packPath)

	return len(entries)
}

func detectNextSeq(dir, prefix string) int {
	entries, _ := os.ReadDir(dir)
	max := 0
	pattern := prefix + "_%04d.pack"
	for _, e := range entries {
		var seq int
		if _, err := fmt.Sscanf(e.Name(), pattern, &seq); err == nil {
			if seq >= max {
				max = seq + 1
			}
		}
	}
	return max
}

func fmtSize(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return "?"
	}
	return fmt.Sprintf("%dMB", fi.Size()/(1024*1024))
}

func readHashList(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var hashes []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		h := scanner.Text()
		if len(h) == hashLen {
			hashes = append(hashes, h)
		}
	}
	return hashes, scanner.Err()
}
