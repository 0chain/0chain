package common

import (
	"bytes"
	"compress/zlib"
	"io"

	"github.com/golang/snappy"
	"github.com/valyala/gozstd"
)

// CompDe - an interface that provides compression/decompression
type CompDe interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
	Encoding() string
}

// SnappyCompDe - a CompDe baseed on Snappy
type SnappyCompDe struct {
}

// NewSnappyCompDe - create a new SnappyCompDe object
func NewSnappyCompDe() *SnappyCompDe {
	return &SnappyCompDe{}
}

// Compress -implement interface
func (scd *SnappyCompDe) Compress(data []byte) []byte {
	return snappy.Encode(nil, data)
}

// Decompress - implement interface
func (scd *SnappyCompDe) Decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}

// Encoding - implement interface
func (scd *SnappyCompDe) Encoding() string {
	return "snappy"
}

// ZStdCompDe - a CompDe based on zstandard
type ZStdCompDe struct {
	level int
}

// NewZStdCompDe - create a new ZStdCompDe object
func NewZStdCompDe() *ZStdCompDe {
	return &ZStdCompDe{}
}

// SetLevel - set the level of compression. 0 = default from the underlying library
func (zstd *ZStdCompDe) SetLevel(level int) {
	zstd.level = level
}

// Compress - implement interface
func (zstd *ZStdCompDe) Compress(data []byte) ([]byte, error) {
	if zstd.level == 0 {
		return gozstd.Compress(nil, data), nil
	} else {
		return gozstd.CompressLevel(nil, data, zstd.level), nil
	}
}

// Decompress - implement interface
func (zstd *ZStdCompDe) Decompress(data []byte) ([]byte, error) {
	return gozstd.Decompress(nil, data)
}

// Encoding - implement interface
func (zstd *ZStdCompDe) Encoding() string {
	return "zstd"
}

// ZStdDictCompDe - a CompDe using dictionary based on zstandard
type ZStdDictCompDe struct {
	cdict *gozstd.CDict
	ddict *gozstd.DDict
}

// NewZStdCompDeWithDict - create a new ZStdDictCompDe
func NewZStdCompDeWithDict(dict []byte) (*ZStdDictCompDe, error) {
	cdict, err := gozstd.NewCDict(dict)
	if err != nil {
		return nil, err
	}
	ddict, err := gozstd.NewDDict(dict)
	if err != nil {
		return nil, err
	}
	cd := &ZStdDictCompDe{}
	cd.cdict = cdict
	cd.ddict = ddict
	return cd, nil
}

// Compress - implement interface
func (zstd *ZStdDictCompDe) Compress(data []byte) []byte {
	return gozstd.CompressDict(nil, data, zstd.cdict)
}

// Decompress - implement interface
func (zstd *ZStdDictCompDe) Decompress(data []byte) ([]byte, error) {
	return gozstd.DecompressDict(nil, data, zstd.ddict)
}

// Encoding - implement interface
func (zstd *ZStdDictCompDe) Encoding() string {
	return "zstddict"
}

// ZLibCompDe - a CompDe based on zlib
type ZLibCompDe struct {
}

// NewZLibCompDe - create a new ZLibCompDe object
func NewZLibCompDe() *ZLibCompDe {
	return &ZLibCompDe{}
}

// Compress - implement interface
func (zlibcd *ZLibCompDe) Compress(data []byte) ([]byte, error) {
	bf := bytes.NewBuffer(nil)
	w, err := zlib.NewWriterLevel(bf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(data); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return bf.Bytes(), nil
}

// Decompress - implement interface
func (zlibcd *ZLibCompDe) Decompress(data []byte) ([]byte, error) {
	reader := bytes.NewBuffer(data)
	r, err := zlib.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	bf := bytes.NewBuffer(nil)
	if _, err := io.Copy(bf, r); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

// Encoding - implement interface
func (zlibcd *ZLibCompDe) Encoding() string {
	return "zlib"
}
