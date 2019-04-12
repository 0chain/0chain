package blockstore

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"

	"0chain.net/chaincore/chain"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

/*FSBlockStore - a block store implementation using file system */
type FSBlockStore struct {
	RootDirectory         string
	blockMetadataProvider datastore.EntityMetadata
}

/*NewFSBlockStore - return a new fs block store */
func NewFSBlockStore(rootDir string) *FSBlockStore {
	store := &FSBlockStore{RootDirectory: rootDir}
	store.blockMetadataProvider = datastore.GetEntityMetadata("block")
	return store
}

func (fbs *FSBlockStore) getFileWithoutExtension(hash string, round int64) string {
	var file bytes.Buffer
	var dirRoundRange = chain.GetServerChain().RoundRange
	fmt.Fprintf(&file, "%s%s%v", fbs.RootDirectory, string(os.PathSeparator), round/dirRoundRange)
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&file, "%s%s", string(os.PathSeparator), hash[3*i:3*i+3])
	}
	fmt.Fprintf(&file, "%s%s", string(os.PathSeparator), hash[9:])
	return file.String()
}

func (fbs *FSBlockStore) getFileName(hash string, round int64) string {
	return fbs.getFileWithoutExtension(hash, round) + ".dat.zlib"
}

/*Write - write the block to the file system */
func (fbs *FSBlockStore) Write(b *block.Block) error {
	fileName := fbs.getFileName(b.Hash, b.Round)
	dir := filepath.Dir(fileName)
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	bf := bufio.NewWriterSize(f, 64*1024)
	w, _ := zlib.NewWriterLevel(bf, zlib.BestCompression)
	datastore.WriteJSON(w, b)
	w.Close()
	bf.Flush()
	f.Close()
	return nil
}

/*ReadWithBlockSummary - read the block given the block summary */
func (fbs *FSBlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	return fbs.read(bs.Hash, bs.Round)
}

/*Read - read the block from the file system */
func (fbs *FSBlockStore) Read(hash string) (*block.Block, error) {
	return nil, common.NewError("interface_not_implemented", "FSBlockStore cannot provide this interface")
}

func (fbs *FSBlockStore) read(hash string, round int64) (*block.Block, error) {
	if len(hash) != 64 {
		return nil, encryption.ErrInvalidHash
	}
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
	var b block.Block
	err = datastore.ReadJSON(r, &b)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

/*Delete - delete from the hash of the block*/
func (fbs *FSBlockStore) Delete(hash string) error {
	return common.NewError("interface_not_implemented", "FSBlockStore cannote provide this interface")
}

/*DeleteBlock - delete the given block from the file system */
func (fbs *FSBlockStore) DeleteBlock(b *block.Block) error {
	fileName := fbs.getFileName(b.Hash, b.Round)
	err := os.Remove(fileName)
	if err != nil {
		return err
	}
	return nil
}
