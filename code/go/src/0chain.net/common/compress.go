package common

import (
	"github.com/golang/snappy"
	"github.com/valyala/gozstd"
)

//CompDe - an interface that provides compression/decompression
type CompDe interface {
	Compress([]byte) []byte
	Decompress([]byte) ([]byte, error)
}

//SnappyCompDe - a CompDe baseed on Snappy
type SnappyCompDe struct {
}

//NewSnappyCompDe - create a new SnappyCompDe object
func NewSnappyCompDe() *SnappyCompDe {
	return &SnappyCompDe{}
}

//Compress -implement interface
func (scd *SnappyCompDe) Compress(data []byte) []byte {
	return snappy.Encode(nil, data)
}

//Decompress - implement interface
func (scd *SnappyCompDe) Decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}

//ZStdCompDe - a CompDe based on zstandard
type ZStdCompDe struct {
}

//NewZStdCompDe - create a new ZStdCompDe object
func NewZStdCompDe() *ZStdCompDe {
	return &ZStdCompDe{}
}

//Compress - implement interface
func (zstd *ZStdCompDe) Compress(data []byte) []byte {
	return gozstd.Compress(nil, data)
}

//Decompress - implement interface
func (zstd *ZStdCompDe) Decompress(data []byte) ([]byte, error) {
	return gozstd.Decompress(nil, data)
}
