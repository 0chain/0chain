// +build ignore

// blockcompact v12: inode-sorted compactor with scan file reuse.
// Skips scanning if scan_<prefix>.txt already exists.
//
// Usage:
//   blockcompact_v12 -blocks /path/to/blocks -prefix 0 -n 1000 -batch 10000
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	v12PackMagic      = "BPAK"
	v12HashLen        = 64
	v12IndexEntrySize = v12HashLen + 8 + 4
	v12TrailerSize    = 16
	v12Extension      = "dat.zlib"
	v12SubDirs        = 5
)

type v12IndexEntry struct {
	hash   string
	offset uint64
	length uint32
}

type v12BlockData struct {
	hash string
	data []byte
}

func main() {
	blocksDir := flag.String("blocks", "", "block store base path (required)")
	prefix := flag.String("prefix", "", "hex prefix 0-f (required)")
	maxPacks := flag.Int("n", 0, "max packs to create (0=unlimited)")
	batchSize := flag.Int("batch", 10000, "blocks per pack")
	flag.Parse()

	if *blocksDir == "" || *prefix == "" {
		fmt.Fprintln(os.Stderr, "error: -blocks and -prefix required")
		flag.Usage()
		os.Exit(1)
	}

	packsDir := filepath.Join(*blocksDir, "packs")
	os.MkdirAll(packsDir, 0700)

	scanFile := filepath.Join(packsDir, fmt.Sprintf("scan_%s.txt", *prefix))
	prefixDir := filepath.Join(*blocksDir, *prefix)

	// Phase 1: scan or reuse
	prescanFile := filepath.Join(packsDir, fmt.Sprintf("prescan_%s.txt", *prefix))
	if fi, err := os.Stat(prescanFile); err == nil && fi.Size() > 0 {
		os.Rename(prescanFile, scanFile)
		fmt.Fprintf(os.Stderr, "Phase 1: using prescan file (%d bytes)\n", fi.Size())
	}
	if fi, err := os.Stat(scanFile); err == nil && fi.Size() > 0 {
		fmt.Fprintf(os.Stderr, "Phase 1: reusing existing scan file %s (%d bytes)\n", scanFile, fi.Size())
	} else {
		fmt.Fprintf(os.Stderr, "Phase 1: scanning %s...\n", prefixDir)
		cmd := exec.Command("bash", "-c",
			fmt.Sprintf("find %s -name '*.%s' -printf '%%i %%p\\n' | sort -n > %s",
				prefixDir, v12Extension, scanFile))
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
			os.Exit(1)
		}
	}

	// Phase 2: read scan file and pack
	f, err := os.Open(scanFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening scan file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	start := time.Now()
	seq := v12DetectNextSeq(packsDir, *prefix)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var batch []v12BlockData
	totalPacked := 0
	totalSkipped := 0
	totalFiles := 0
	packCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		// Format: "inode path"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		path := parts[1]

		hash := v12ExtractHash(*blocksDir, path)
		if hash == "" {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			totalSkipped++
			continue
		}

		batch = append(batch, v12BlockData{hash: hash, data: data})
		totalFiles++

		if len(batch) >= *batchSize {
			packPath := filepath.Join(packsDir, fmt.Sprintf("inode_%s_%04d.pack", *prefix, seq))
			n := v12WritePack(batch, packPath)
			totalPacked += n
			packCount++
			elapsed := time.Since(start).Round(time.Second)
			rate := 0
			if elapsed.Seconds() > 0 {
				rate = int(float64(totalPacked) / elapsed.Seconds())
			}
			fmt.Fprintf(os.Stderr, "  pack %d: %d blocks, %s, %d blk/s, %d skipped\n",
				seq, n, v12FmtSize(packPath), rate, totalSkipped)
			seq++
			batch = batch[:0]

			if *maxPacks > 0 && packCount >= *maxPacks {
				break
			}
		}
	}

	// Flush remaining
	if len(batch) > 0 {
		packPath := filepath.Join(packsDir, fmt.Sprintf("inode_%s_%04d.pack", *prefix, seq))
		n := v12WritePack(batch, packPath)
		totalPacked += n
		fmt.Fprintf(os.Stderr, "  pack %d: %d blocks (final), %s\n", seq, n, v12FmtSize(packPath))
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Fprintf(os.Stderr, "Done. %d packed, %d skipped, %d files in %v\n",
		totalPacked, totalSkipped, totalFiles, elapsed)
}

func v12ExtractHash(blocksDir, fullPath string) string {
	rel, err := filepath.Rel(blocksDir, fullPath)
	if err != nil {
		return ""
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) < v12SubDirs+1 {
		return ""
	}
	hash := ""
	for i := 0; i < v12SubDirs; i++ {
		if len(parts[i]) != 1 {
			return ""
		}
		hash += parts[i]
	}
	filename := parts[v12SubDirs]
	rest := strings.TrimSuffix(filename, "."+v12Extension)
	hash += rest
	if len(hash) == v12HashLen {
		return hash
	}
	return ""
}

func v12WritePack(blocks []v12BlockData, packPath string) int {
	tmpPath := packPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create error: %v\n", err)
		return 0
	}

	var entries []v12IndexEntry
	var offset uint64

	for _, b := range blocks {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(b.data)))
		f.Write(lenBuf[:])
		f.Write(b.data)
		entries = append(entries, v12IndexEntry{
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
		var buf [v12IndexEntrySize]byte
		copy(buf[:v12HashLen], e.hash)
		binary.LittleEndian.PutUint64(buf[v12HashLen:], e.offset)
		binary.LittleEndian.PutUint32(buf[v12HashLen+8:], e.length)
		f.Write(buf[:])
	}

	var trailer [v12TrailerSize]byte
	binary.LittleEndian.PutUint64(trailer[:8], indexOffset)
	binary.LittleEndian.PutUint32(trailer[8:12], uint32(len(entries)))
	copy(trailer[12:], v12PackMagic)
	f.Write(trailer[:])

	f.Sync()
	f.Close()
	os.Rename(tmpPath, packPath)
	return len(entries)
}

func v12DetectNextSeq(packsDir, prefix string) int {
	entries, _ := os.ReadDir(packsDir)
	max := 0
	pattern := "inode_" + prefix + "_%04d.pack"
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

func v12FmtSize(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return "?"
	}
	return fmt.Sprintf("%dMB", fi.Size()/(1024*1024))
}
