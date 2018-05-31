package block

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"

	"0chain.net/datastore"
)

/*BlockStore - an interface to read and write blocks to some storage */
type BlockStore interface {
	Write(b *Block) error
	Read(hash string, round int64) (*Block, error)
}

/*FileBlockStore - a block store implementation using file system */
type FileBlockStore struct {
	RootDirectory string
}

const (
	DIR_ROUND_RANGE = 10000000
)

var Store BlockStore

/*SetupFileBlockStore - Setup a file system based block storage */
func SetupFileBlockStore(rootDir string) {
	Store = &FileBlockStore{RootDirectory: rootDir}
}

func (fbs *FileBlockStore) getFileName(hash string, round int64) string {
	var dir bytes.Buffer
	fmt.Fprintf(&dir, "%v%s%v", fbs.RootDirectory, string(os.PathSeparator), int64(round/DIR_ROUND_RANGE))
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&dir, "%s%v", string(os.PathSeparator), hash[i:i+8])
	}
	fmt.Fprintf(&dir, ".dat.zlib")
	return dir.String()
}

/*GetFileName - given a block, get the file name it maps to */
func (fbs *FileBlockStore) GetFileName(b *Block) string {
	return fbs.getFileName(b.Hash, b.Round)
}

/*Write - write the block to the file system */
func (fbs *FileBlockStore) Write(b *Block) error {
	fileName := fbs.GetFileName(b)
	dir := filepath.Dir(fileName)
	//file := filepath.Base(fileName)
	os.MkdirAll(dir, 0777)
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zlib.NewWriter(f)
	defer w.Close()
	datastore.WriteJSON(w, b)
	return nil
}

/*Read - read the block from the file system */
func (fbs *FileBlockStore) Read(hash string, round int64) (*Block, error) {
	fileName := fbs.getFileName(hash, round)
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var b Block
	err = datastore.ReadJSON(r, &b)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
