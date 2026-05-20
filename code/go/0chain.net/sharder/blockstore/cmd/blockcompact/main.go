// blockcompact packs loose block files into pack files.
// It reads files in inode order (physical disk order) for maximum HDD throughput,
// then builds sorted hash indices for fast lookup.
//
// Usage:
//   blockcompact -blocks /path/to/blocks [-n 10] [-batch 10000]
//
// Phase 1: scan directory tree, collect file paths + inodes (single traversal)
// Phase 2: sort by inode, read in batches of -batch files, write pack files
// Pack files are named by sequence number. Each contains a sorted hash index.
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	packMagic      = "BPAK"
	hashLen        = 64
	indexEntrySize = hashLen + 8 + 4
	trailerSize    = 8 + 4 + 4
	extension      = "dat.zlib"
	subDirs        = 5
)

type indexEntry struct {
	hash   string
	offset uint64
	length uint32
}

type inodeFile struct {
	inode uint64
	path  string
	hash  string
}

func main() {
	blocksDir := flag.String("blocks", "", "path to blocks directory (required)")
	numPacks := flag.Int("n", 10, "number of packs to create")
	batchSize := flag.Int("batch", 10000, "blocks per pack")
	prefix := flag.String("prefix", "", "only scan this hex prefix subdir (0-f)")
	flag.Parse()

	if *blocksDir == "" {
		fmt.Fprintln(os.Stderr, "error: -blocks flag is required")
		flag.Usage()
		os.Exit(1)
	}

	packsDir := filepath.Join(*blocksDir, "packs")
	os.MkdirAll(packsDir, 0700)

	scanDir := *blocksDir
	if *prefix != "" {
		scanDir = filepath.Join(*blocksDir, *prefix)
	}

	// Phase 1: single full scan, sorted by inode, saved to file
	scanFile := filepath.Join(packsDir, "scan.txt")
	if *prefix != "" {
		scanFile = filepath.Join(packsDir, fmt.Sprintf("scan_%s.txt", *prefix))
	}

	fmt.Printf("Phase 1: scanning all files in %s by inode...\n", scanDir)
	scanStart := time.Now()
	cmdStr := fmt.Sprintf(
		"find %s -name '*.%s' -not -path '*/packs/*' -printf '%%i %%p\\n' | sort -n > %s",
		scanDir, extension, scanFile)
	cmd := exec.Command("bash", "-c", cmdStr)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %s %v\n", string(out), err)
		os.Exit(1)
	}

	// Count lines
	countCmd := exec.Command("wc", "-l", scanFile)
	countOut, _ := countCmd.Output()
	totalFiles := 0
	fmt.Sscanf(string(countOut), "%d", &totalFiles)
	fmt.Printf("  scanned %d files in %v\n", totalFiles, time.Since(scanStart).Round(time.Second))

	// Phase 2: read batches from the sorted file
	sf, err := os.Open(scanFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open scan file: %v\n", err)
		os.Exit(1)
	}
	defer sf.Close()
	scanner := bufio.NewScanner(sf)

	seq := detectNextSeq(packsDir)
	totalPacked := 0

	for i := 0; i < *numPacks; i++ {
		fmt.Printf("[%d/%d] reading %d files from scan...\n", i+1, *numPacks, *batchSize)
		batchStart := time.Now()
		var files []inodeFile
		for scanner.Scan() && len(files) < *batchSize {
			line := scanner.Text()
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}
			inode, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				continue
			}
			hash := extractHash(*blocksDir, parts[1])
			if hash == "" {
				continue
			}
			files = append(files, inodeFile{inode: inode, path: parts[1], hash: hash})
		}
		scanTime := time.Since(batchStart).Round(time.Second)

		if len(files) < *batchSize {
			fmt.Printf("  only %d files (need %d), done.\n", len(files), *batchSize)
			break
		}

		packStart := time.Now()
		packName := fmt.Sprintf("inode_%06d.pack", seq)
		if *prefix != "" {
			packName = fmt.Sprintf("inode_%s_%04d.pack", *prefix, seq)
		}
		packPath := filepath.Join(packsDir, packName)

		n, err := packBatch(files, packPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pack error: %v\n", err)
			os.Exit(1)
		}

		packTime := time.Since(packStart).Round(time.Second)
		fi, _ := os.Stat(packPath)
		rate := 0
		if packTime.Seconds() > 0 {
			rate = int(float64(n) / packTime.Seconds())
		}
		fmt.Printf("  pack %d: %d blocks, %dMB, scan=%v pack=%v (%d blk/s)\n",
			seq, n, fi.Size()/(1024*1024), scanTime, packTime, rate)

		seq++
		totalPacked += n
	}

	fmt.Printf("Done. %d blocks packed into %d files.\n", totalPacked, seq-detectNextSeq(packsDir)+int(totalPacked/(*batchSize)))
}

// scanByInodeAfter scans files with inode > afterInode, sorted by inode.
func scanByInodeAfter(blocksDir, scanDir string, limit int, afterInode uint64) ([]inodeFile, error) {
	var cmdStr string
	if afterInode == 0 {
		cmdStr = fmt.Sprintf(
			"find %s -name '*.%s' -not -path '*/packs/*' -printf '%%i %%p\\n' | head -%d",
			scanDir, extension, limit)
	} else {
		cmdStr = fmt.Sprintf(
			"find %s -name '*.%s' -not -path '*/packs/*' -printf '%%i %%p\\n' | awk '$1 > %d' | head -%d",
			scanDir, extension, afterInode, limit)
	}

	cmd := exec.Command("bash", "-c", cmdStr)
	out, err := cmd.Output()
	if err != nil {
		// head causes find to get SIGPIPE which is exit 141, that's OK
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 141 {
			// normal — head closed pipe
		} else if len(out) == 0 {
			return nil, fmt.Errorf("find failed: %w", err)
		}
	}

	var files []inodeFile
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		inode, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			continue
		}
		path := parts[1]
		hash := extractHash(blocksDir, path)
		if hash == "" {
			continue
		}
		files = append(files, inodeFile{inode: inode, path: path, hash: hash})
	}

	// Sort by inode = physical disk order
	sort.Slice(files, func(i, j int) bool {
		return files[i].inode < files[j].inode
	})

	return files, nil
}

func packBatch(files []inodeFile, packPath string) (int, error) {
	tmpPath := packPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return 0, err
	}

	var entries []indexEntry
	var offset uint64
	packed := 0

	for _, inf := range files {
		data, err := readFile(inf.path)
		if err != nil {
			continue
		}

		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(data)))
		f.Write(lenBuf[:])
		f.Write(data)

		entries = append(entries, indexEntry{
			hash:   inf.hash,
			offset: offset + 4,
			length: uint32(len(data)),
		})
		offset += 4 + uint64(len(data))
		packed++
	}

	if packed == 0 {
		f.Close()
		os.Remove(tmpPath)
		return 0, nil
	}

	// Sort index by hash for binary search
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

	if err := os.Rename(tmpPath, packPath); err != nil {
		return 0, err
	}

	// Verify
	if err := verifyPack(packPath); err != nil {
		fmt.Fprintf(os.Stderr, "  WARNING: verify failed: %v\n", err)
	}

	return packed, nil
}

func detectNextSeq(packsDir string) int {
	entries, _ := os.ReadDir(packsDir)
	max := 0
	for _, e := range entries {
		var seq int
		if _, err := fmt.Sscanf(e.Name(), "inode_%06d.pack", &seq); err == nil {
			if seq >= max {
				max = seq + 1
			}
		}
	}
	return max
}

func verifyPack(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	var trailer [trailerSize]byte
	if _, err := f.ReadAt(trailer[:], fi.Size()-int64(trailerSize)); err != nil {
		return err
	}
	if string(trailer[12:16]) != packMagic {
		return fmt.Errorf("invalid magic")
	}

	indexOffset := binary.LittleEndian.Uint64(trailer[:8])
	indexCount := binary.LittleEndian.Uint32(trailer[8:12])

	indexBuf := make([]byte, int64(indexCount)*int64(indexEntrySize))
	if _, err := f.ReadAt(indexBuf, int64(indexOffset)); err != nil {
		return fmt.Errorf("read index: %w", err)
	}

	var prev string
	for i := uint32(0); i < indexCount; i++ {
		off := int64(i) * int64(indexEntrySize)
		hash := string(bytes.TrimRight(indexBuf[off:off+hashLen], "\x00"))
		if hash < prev {
			return fmt.Errorf("index not sorted at entry %d", i)
		}
		prev = hash
	}

	return nil
}

func extractHash(basePath, fullPath string) string {
	rel, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return ""
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) < subDirs+1 {
		return ""
	}
	hash := ""
	for i := 0; i < subDirs; i++ {
		if len(parts[i]) != 1 {
			return ""
		}
		hash += parts[i]
	}
	rest := strings.TrimSuffix(parts[subDirs], "."+extension)
	hash += rest
	return hash
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
