package blockstore

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// Global index format: sorted array of {hash[64], packID[2]}
// packID maps to a pack file path via a header table.
// File layout:
//   [header: packCount[2] + packCount × pathLen[2] + path[variable]]
//   [entries: sorted by hash, each = hash[64] + packID[2]]
//   [trailer: entryCount[4] + headerSize[4] + magic "GIDX"[4]]

const (
	globalIdxMagic    = "GIDX"
	globalEntrySize   = hashLen + 2 // hash + packID
	globalTrailerSize = 4 + 4 + 4   // entryCount + headerSize + magic
)

// globalIndex provides O(log N) hash→pack lookup via a sorted binary file.
type globalIndex struct {
	data      []byte   // mmap'd or loaded file contents
	packPaths []string // packID → file path
	entryOff  int      // byte offset where entries start
	count     int      // number of entries
}

// loadGlobalIndex loads the global index from disk using mmap for zero-copy access.
func loadGlobalIndex(packsDir string) (*globalIndex, error) {
	path := filepath.Join(packsDir, "global.idx")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int(fi.Size())
	if size < globalTrailerSize {
		return nil, fmt.Errorf("global index too small")
	}

	// mmap the file for zero-copy access
	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("mmap global index: %w", err)
	}

	// Read trailer
	tOff := len(data) - globalTrailerSize
	entryCount := int(binary.LittleEndian.Uint32(data[tOff:]))
	headerSize := int(binary.LittleEndian.Uint32(data[tOff+4:]))
	magic := string(data[tOff+8 : tOff+12])
	if magic != globalIdxMagic {
		return nil, fmt.Errorf("invalid global index magic")
	}

	// Read header (pack paths)
	if headerSize > len(data) {
		return nil, fmt.Errorf("invalid header size")
	}
	packCount := int(binary.LittleEndian.Uint16(data[0:2]))
	packPaths := make([]string, packCount)
	off := 2
	for i := 0; i < packCount; i++ {
		pathLen := int(binary.LittleEndian.Uint16(data[off:]))
		off += 2
		packPaths[i] = string(data[off : off+pathLen])
		off += pathLen
	}

	return &globalIndex{
		data:      data,
		packPaths: packPaths,
		entryOff:  headerSize,
		count:     entryCount,
	}, nil
}

// lookup returns the pack file path containing the given hash, or "" if not found.
func (gi *globalIndex) lookup(hash string) string {
	lo, hi := 0, gi.count-1
	for lo <= hi {
		mid := (lo + hi) / 2
		off := gi.entryOff + mid*globalEntrySize
		entryHash := string(gi.data[off : off+hashLen])
		if entryHash == hash {
			packID := int(binary.LittleEndian.Uint16(gi.data[off+hashLen:]))
			if packID < len(gi.packPaths) {
				return gi.packPaths[packID]
			}
			return ""
		}
		if entryHash < hash {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return ""
}

// buildGlobalIndex scans all pack files and creates global.idx.
func buildGlobalIndex(packsDir string) error {
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return err
	}

	type gEntry struct {
		hash   string
		packID uint16
	}

	var packPaths []string
	var allEntries []gEntry

	for _, e := range entries {
		if e.IsDir() || len(e.Name()) < 5 {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".pack" {
			continue
		}

		path := filepath.Join(packsDir, e.Name())
		pr, err := openPackIndex(path)
		if err != nil {
			continue
		}

		packID := uint16(len(packPaths))
		packPaths = append(packPaths, e.Name())

		for _, pe := range pr.entries {
			allEntries = append(allEntries, gEntry{hash: pe.Hash, packID: packID})
		}

		logging.Logger.Debug("global index: processed pack",
			zap.String("pack", e.Name()),
			zap.Int("blocks", len(pr.entries)))
	}

	// Sort by hash
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].hash < allEntries[j].hash
	})

	// Write file
	outPath := filepath.Join(packsDir, "global.idx")
	f, err := os.Create(outPath + ".tmp")
	if err != nil {
		return err
	}

	// Write header: packCount + pack paths
	headerStart := 0
	binary.Write(f, binary.LittleEndian, uint16(len(packPaths)))
	for _, p := range packPaths {
		binary.Write(f, binary.LittleEndian, uint16(len(p)))
		f.Write([]byte(p))
	}
	headerEnd, _ := f.Seek(0, 1)

	// Write entries
	for _, e := range allEntries {
		var buf [globalEntrySize]byte
		copy(buf[:hashLen], e.hash)
		binary.LittleEndian.PutUint16(buf[hashLen:], e.packID)
		f.Write(buf[:])
	}

	// Write trailer
	var trailer [globalTrailerSize]byte
	binary.LittleEndian.PutUint32(trailer[:4], uint32(len(allEntries)))
	binary.LittleEndian.PutUint32(trailer[4:8], uint32(headerEnd-int64(headerStart)))
	copy(trailer[8:], globalIdxMagic)
	f.Write(trailer[:])

	f.Sync()
	f.Close()

	if err := os.Rename(outPath+".tmp", outPath); err != nil {
		return err
	}

	logging.Logger.Info("global index built",
		zap.Int("packs", len(packPaths)),
		zap.Int("blocks", len(allEntries)),
		zap.String("path", outPath))

	return nil
}
