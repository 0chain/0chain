package blockstore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

// Pack file format:
//   [data section: concatenated length-prefixed block blobs]
//   [index section: sorted entries of {hash[64], offset[8], length[4]}]
//   [trailer: indexOffset[8] + indexCount[4] + magic[4]]
//
// Trailer is at the end so the file can be written sequentially:
// append blocks, then append index, then append trailer.

const (
	packMagic      = "BPAK"
	packVersion    = 1
	hashLen        = 64              // hex-encoded SHA256
	indexEntrySize = hashLen + 8 + 4 // hash + offset + length
	trailerSize    = 8 + 4 + 4       // indexOffset + indexCount + magic
	blocksPerPack  = 10000
)

// packIndexEntry is a single entry in the pack index.
type packIndexEntry struct {
	Hash   string // 64-char hex hash
	Offset uint64 // byte offset in pack file where block data starts
	Length uint32 // byte length of block data
}

// packWriter creates a new pack file by appending blocks then writing the index.
type packWriter struct {
	f       *os.File
	entries []packIndexEntry
	offset  uint64
}

func newPackWriter(path string) (*packWriter, error) {
	f, err := os.Create(path + ".tmp")
	if err != nil {
		return nil, err
	}
	return &packWriter{f: f}, nil
}

// addBlock appends a raw block blob (already zlib-compressed) to the pack.
func (pw *packWriter) addBlock(hash string, data []byte) error {
	// Write length prefix + data
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := pw.f.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := pw.f.Write(data); err != nil {
		return err
	}

	pw.entries = append(pw.entries, packIndexEntry{
		Hash:   hash,
		Offset: pw.offset + 4, // skip the length prefix
		Length: uint32(len(data)),
	})
	pw.offset += 4 + uint64(len(data))
	return nil
}

// finish sorts the index, writes it, writes the trailer, and renames to final path.
func (pw *packWriter) finish(finalPath string) error {
	// Sort entries by hash for binary search
	sort.Slice(pw.entries, func(i, j int) bool {
		return pw.entries[i].Hash < pw.entries[j].Hash
	})

	indexOffset := pw.offset

	// Write index entries
	for _, e := range pw.entries {
		var buf [indexEntrySize]byte
		copy(buf[:hashLen], e.Hash)
		binary.LittleEndian.PutUint64(buf[hashLen:], e.Offset)
		binary.LittleEndian.PutUint32(buf[hashLen+8:], e.Length)
		if _, err := pw.f.Write(buf[:]); err != nil {
			return err
		}
	}

	// Write trailer
	var trailer [trailerSize]byte
	binary.LittleEndian.PutUint64(trailer[:8], indexOffset)
	binary.LittleEndian.PutUint32(trailer[8:12], uint32(len(pw.entries)))
	copy(trailer[12:], packMagic)
	if _, err := pw.f.Write(trailer[:]); err != nil {
		return err
	}

	if err := pw.f.Sync(); err != nil {
		return err
	}
	if err := pw.f.Close(); err != nil {
		return err
	}

	return os.Rename(finalPath+".tmp", finalPath)
}

func (pw *packWriter) abort() {
	pw.f.Close()
	os.Remove(pw.f.Name())
}

func (pw *packWriter) count() int {
	return len(pw.entries)
}

// packReader reads blocks from a pack file using its embedded index.
type packReader struct {
	path    string
	entries []packIndexEntry // sorted by hash
}

// openPackIndex reads the index from a pack file.
func openPackIndex(path string) (*packReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read trailer from end of file
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() < int64(trailerSize) {
		return nil, fmt.Errorf("pack file too small: %s", path)
	}

	var trailer [trailerSize]byte
	if _, err := f.ReadAt(trailer[:], fi.Size()-int64(trailerSize)); err != nil {
		return nil, err
	}

	magic := string(trailer[12:16])
	if magic != packMagic {
		return nil, fmt.Errorf("invalid pack magic: %s", magic)
	}

	indexOffset := binary.LittleEndian.Uint64(trailer[:8])
	indexCount := binary.LittleEndian.Uint32(trailer[8:12])

	// Read index section
	indexSize := int64(indexCount) * int64(indexEntrySize)
	indexBuf := make([]byte, indexSize)
	if _, err := f.ReadAt(indexBuf, int64(indexOffset)); err != nil {
		return nil, err
	}

	entries := make([]packIndexEntry, indexCount)
	for i := uint32(0); i < indexCount; i++ {
		off := int64(i) * int64(indexEntrySize)
		entries[i] = packIndexEntry{
			Hash:   string(bytes.TrimRight(indexBuf[off:off+hashLen], "\x00")),
			Offset: binary.LittleEndian.Uint64(indexBuf[off+hashLen:]),
			Length: binary.LittleEndian.Uint32(indexBuf[off+hashLen+8:]),
		}
	}

	return &packReader{path: path, entries: entries}, nil
}

// lookup finds a block by hash using binary search. Returns raw compressed bytes.
func (pr *packReader) lookup(hash string) ([]byte, error) {
	idx := sort.Search(len(pr.entries), func(i int) bool {
		return pr.entries[i].Hash >= hash
	})
	if idx >= len(pr.entries) || pr.entries[idx].Hash != hash {
		return nil, nil // not found
	}

	e := pr.entries[idx]
	f, err := os.Open(pr.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data := make([]byte, e.Length)
	if _, err := f.ReadAt(data, int64(e.Offset)); err != nil {
		return nil, err
	}
	return data, nil
}

// minHash returns the smallest hash in this pack.
func (pr *packReader) minHash() string {
	if len(pr.entries) == 0 {
		return ""
	}
	return pr.entries[0].Hash
}

// maxHash returns the largest hash in this pack.
func (pr *packReader) maxHash() string {
	if len(pr.entries) == 0 {
		return ""
	}
	return pr.entries[len(pr.entries)-1].Hash
}

// readRawBlock reads a raw block file from disk (the zlib-compressed bytes).
func readRawBlock(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
