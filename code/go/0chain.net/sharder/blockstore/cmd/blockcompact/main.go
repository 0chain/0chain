// blockcompact packs loose block files into pack files.
//
// Usage:
//   tar cf - blocks/0/ | blockcompact -packs /path/to/packs -prefix 0 -batch 10000
//
// Reads a tar stream from stdin. Extracts hash from each file path,
// reads block data, packs into 10K-block pack files with sorted hash index.
// Sequential HDD read via tar = ~100MB/s = ~15 min per prefix.
package main

import (
	"archive/tar"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
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
	extension      = "dat.zlib"
	subDirs        = 5
)

type indexEntry struct {
	hash   string
	offset uint64
	length uint32
}

type blockData struct {
	hash string
	data []byte
}

func main() {
	packsDir := flag.String("packs", "", "output packs directory (required)")
	prefix := flag.String("prefix", "", "prefix label for pack names")
	batchSize := flag.Int("batch", 10000, "blocks per pack")
	flag.Parse()

	if *packsDir == "" {
		fmt.Fprintln(os.Stderr, "error: -packs flag required")
		flag.Usage()
		os.Exit(1)
	}
	os.MkdirAll(*packsDir, 0700)

	tr := tar.NewReader(os.Stdin)
	seq := detectNextSeq(*packsDir, *prefix)
	var batch []blockData
	totalPacked := 0
	totalFiles := 0
	start := time.Now()

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar read error: %v\n", err)
			break
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if !strings.HasSuffix(hdr.Name, "."+extension) {
			continue
		}

		hash := extractHash(hdr.Name)
		if hash == "" {
			continue
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			continue
		}

		batch = append(batch, blockData{hash: hash, data: data})
		totalFiles++

		if len(batch) >= *batchSize {
			packPath := packName(*packsDir, *prefix, seq)
			n := writePack(batch, packPath)
			elapsed := time.Since(start).Round(time.Second)
			rate := 0
			if elapsed.Seconds() > 0 {
				rate = int(float64(totalPacked+n) / elapsed.Seconds())
			}
			fmt.Printf("  pack %d: %d blocks, %s, %d blk/s total\n",
				seq, n, fmtSize(packPath), rate)
			totalPacked += n
			seq++
			batch = batch[:0]
		}
	}

	// Flush remaining
	if len(batch) >= *batchSize/2 {
		packPath := packName(*packsDir, *prefix, seq)
		n := writePack(batch, packPath)
		totalPacked += n
		fmt.Printf("  pack %d: %d blocks (final), %s\n", seq, n, fmtSize(packPath))
		seq++
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("Done. %d blocks packed from %d files in %v\n", totalPacked, totalFiles, elapsed)
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

	// Sort index by hash
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].hash < entries[j].hash
	})

	// Write index
	indexOffset := offset
	for _, e := range entries {
		var buf [indexEntrySize]byte
		copy(buf[:hashLen], e.hash)
		binary.LittleEndian.PutUint64(buf[hashLen:], e.offset)
		binary.LittleEndian.PutUint32(buf[hashLen+8:], e.length)
		f.Write(buf[:])
	}

	// Write trailer
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

func extractHash(tarPath string) string {
	// tar path: 0/f/f/c/1/<rest>.dat.zlib (5 single-char dirs + filename)
	parts := strings.Split(tarPath, "/")
	// Find 5 consecutive single-char parts followed by a filename
	for i := 0; i <= len(parts)-subDirs-1; i++ {
		allSingle := true
		for j := 0; j < subDirs; j++ {
			if i+j >= len(parts) || len(parts[i+j]) != 1 {
				allSingle = false
				break
			}
		}
		if !allSingle {
			continue
		}
		// Found 5 single-char dirs starting at i
		hash := ""
		for j := 0; j < subDirs; j++ {
			hash += parts[i+j]
		}
		if i+subDirs >= len(parts) {
			continue
		}
		filename := parts[i+subDirs]
		rest := strings.TrimSuffix(filename, "."+extension)
		hash += rest
		if len(hash) == hashLen {
			return hash
		}
	}
	return ""
}

func packName(packsDir, prefix string, seq int) string {
	if prefix != "" {
		return filepath.Join(packsDir, fmt.Sprintf("pack_%s_%04d.pack", prefix, seq))
	}
	return filepath.Join(packsDir, fmt.Sprintf("pack_%04d.pack", seq))
}

func detectNextSeq(packsDir, prefix string) int {
	entries, _ := os.ReadDir(packsDir)
	max := 0
	pattern := "pack_%04d.pack"
	if prefix != "" {
		pattern = "pack_" + prefix + "_%04d.pack"
	}
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
