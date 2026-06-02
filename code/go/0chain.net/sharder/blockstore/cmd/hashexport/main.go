// hashexport extracts all block hashes from a global.idx file.
// Output: one hash per line, sorted, to stdout.
//
// Usage: hashexport -idx /path/to/packs/global.idx > hashes.txt
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"syscall"
)

const (
	hashLen           = 64
	globalEntrySize   = hashLen + 2
	globalTrailerSize = 12
	globalIdxMagic    = "GIDX"
)

func main() {
	idxPath := flag.String("idx", "", "path to global.idx (required)")
	flag.Parse()

	if *idxPath == "" {
		fmt.Fprintln(os.Stderr, "error: -idx flag required")
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(*idxPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	size := int(fi.Size())
	if size < globalTrailerSize {
		fmt.Fprintln(os.Stderr, "error: file too small")
		os.Exit(1)
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: mmap: %v\n", err)
		os.Exit(1)
	}
	defer syscall.Munmap(data)

	tOff := len(data) - globalTrailerSize
	entryCount := int(binary.LittleEndian.Uint32(data[tOff:]))
	headerSize := int(binary.LittleEndian.Uint32(data[tOff+4:]))
	magic := string(data[tOff+8 : tOff+12])
	if magic != globalIdxMagic {
		fmt.Fprintln(os.Stderr, "error: invalid magic")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Exporting %d hashes...\n", entryCount)

	w := bufio.NewWriterSize(os.Stdout, 256*1024)
	for i := 0; i < entryCount; i++ {
		off := headerSize + i*globalEntrySize
		w.Write(data[off : off+hashLen])
		w.WriteByte('\n')
	}
	w.Flush()

	fmt.Fprintf(os.Stderr, "Done. %d hashes exported.\n", entryCount)
}
