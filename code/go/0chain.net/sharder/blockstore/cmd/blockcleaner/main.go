// blockcleaner deletes loose block files that have been packed into pack files.
// It processes oldest rounds first by reading pack file names (rounds_NNNN_NNNN.pack).
//
// Usage:
//   blockcleaner -blocks /path/to/blocks -n 50000 [-dry-run]
//
// It loads all pack indices, then for each pack (oldest first), finds and deletes
// the corresponding loose files. The -n flag limits total deletions per run.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	hashLen        = 64
	indexEntrySize = hashLen + 8 + 4
	trailerSize    = 8 + 4 + 4
	packMagic      = "BPAK"
	extension      = "dat.zlib"
	subDirs        = 5
)

type indexEntry struct {
	Hash string
}

type packFile struct {
	path    string
	loRound int64
	hiRound int64
	hashes  []string
}

func main() {
	blocksDir := flag.String("blocks", "", "path to blocks directory (required)")
	limit := flag.Int("n", 50000, "max number of loose files to delete per run")
	dryRun := flag.Bool("dry-run", false, "preview deletions without actually deleting")
	flag.Parse()

	if *blocksDir == "" {
		fmt.Fprintln(os.Stderr, "error: -blocks flag is required")
		flag.Usage()
		os.Exit(1)
	}

	packsDir := filepath.Join(*blocksDir, "packs")

	// 1. Load all pack files, sorted by round range (oldest first)
	fmt.Println("Loading pack indices...")
	packs, err := loadPacksSortedByRound(packsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading packs: %v\n", err)
		os.Exit(1)
	}

	totalHashes := 0
	for _, p := range packs {
		totalHashes += len(p.hashes)
	}
	fmt.Printf("Loaded %d packs containing %d blocks (oldest round: %d)\n",
		len(packs), totalHashes, packs[0].loRound)

	// 2. Process packs oldest-first, delete corresponding loose files
	deleted := 0
	errors := 0
	notFound := 0

	for _, pack := range packs {
		if deleted >= *limit {
			break
		}

		for _, hash := range pack.hashes {
			if deleted >= *limit {
				break
			}

			loosePath := hashToLoosePath(*blocksDir, hash)
			if loosePath == "" {
				continue
			}

			_, err := os.Stat(loosePath)
			if os.IsNotExist(err) {
				notFound++
				continue
			}

			if *dryRun {
				deleted++
				continue
			}

			if err := os.Remove(loosePath); err != nil {
				errors++
				if errors <= 5 {
					fmt.Fprintf(os.Stderr, "error deleting %s: %v\n", loosePath, err)
				}
				continue
			}
			deleted++

			// Clean empty parent dirs
			dir := filepath.Dir(loosePath)
			for dir != *blocksDir && dir != "." && dir != "/" {
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) > 0 {
					break
				}
				os.Remove(dir)
				dir = filepath.Dir(dir)
			}

			if deleted%10000 == 0 {
				fmt.Printf("  deleted %d files (pack rounds %d-%d)...\n",
					deleted, pack.loRound, pack.hiRound)
			}
		}
	}

	if *dryRun {
		fmt.Printf("Dry run: would delete %d files (%d already missing)\n", deleted, notFound)
	} else {
		fmt.Printf("Done. Deleted %d files (%d already missing, %d errors)\n",
			deleted, notFound, errors)
	}
}

func loadPacksSortedByRound(packsDir string) ([]packFile, error) {
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read packs directory: %w", err)
	}

	var packs []packFile
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".pack") {
			continue
		}

		var lo, hi int64
		if _, err := fmt.Sscanf(e.Name(), "rounds_%d_%d.pack", &lo, &hi); err != nil {
			// Try legacy sequential naming
			lo, hi = 0, 0
		}

		path := filepath.Join(packsDir, e.Name())
		hashes, err := readPackHashes(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
			continue
		}

		packs = append(packs, packFile{
			path:    path,
			loRound: lo,
			hiRound: hi,
			hashes:  hashes,
		})
	}

	// Sort by lowest round (oldest first)
	sort.Slice(packs, func(i, j int) bool {
		return packs[i].loRound < packs[j].loRound
	})

	if len(packs) == 0 {
		return nil, fmt.Errorf("no pack files found in %s", packsDir)
	}

	return packs, nil
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
		return nil, fmt.Errorf("file too small")
	}

	var trailer [trailerSize]byte
	if _, err := f.ReadAt(trailer[:], fi.Size()-int64(trailerSize)); err != nil {
		return nil, err
	}
	if string(trailer[12:16]) != packMagic {
		return nil, fmt.Errorf("invalid magic")
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

func hashToLoosePath(blocksDir, hash string) string {
	if len(hash) <= subDirs {
		return ""
	}
	var s string
	for i := 0; i < subDirs; i++ {
		s = filepath.Join(s, string(hash[i]))
	}
	return filepath.Join(blocksDir, s, fmt.Sprintf("%s.%s", hash[subDirs:], extension))
}

// walkLooseFiles is unused but kept for reference — counts loose files.
func walkLooseFiles(blocksDir string) int {
	count := 0
	filepath.WalkDir(blocksDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.Name() == "packs" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "."+extension) {
			count++
		}
		return nil
	})
	return count
}
