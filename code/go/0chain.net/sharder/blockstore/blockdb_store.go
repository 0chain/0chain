package blockstore

import (
	"context"
	"io"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/sharder/blockdb"
)

// BlockDBStore is a block store backed by blockdb.
type BlockDBStore struct {
	*FSBlockStore
	txnMetadataProvider datastore.EntityMetadata
	compress            bool
}

var (
	// Make sure BlockDBStore implements BlockStore.
	_ BlockStore = (*BlockDBStore)(nil)
)

// NewBlockDBStore - create a new blockdb store
func NewBlockDBStore(fsbs *FSBlockStore) BlockStore {
	return &BlockDBStore{
		FSBlockStore:        fsbs,
		txnMetadataProvider: datastore.GetEntityMetadata("txn"),
		compress:            true,
	}
}

type blockHeader struct {
	*block.Block
}

var (
	// MakeSure blockHeader implements blockdb.SerDe interface.
	_ blockdb.SerDe = (*blockHeader)(nil)
)

// Encode is a part of blockdb.SerDe interface implementation.
func (bh *blockHeader) Encode(writer io.Writer) error {
	_, err := datastore.ToMsgpack(bh.Block).WriteTo(writer)
	return err
}

// Decode is a part of blockdb.SerDe interface implementation.
func (bh *blockHeader) Decode(reader io.Reader) error {
	return datastore.FromMsgpack(reader, bh.Block)
}

type txnRecord struct {
	*transaction.Transaction
}

var (
	// MakeSure txnRecord implements blockdb.Record interface.
	_ blockdb.Record = (*txnRecord)(nil)
)

// GetKey is a part of blockdb.Record interface implementation.
func (tr *txnRecord) GetKey() blockdb.Key {
	return blockdb.Key(tr.Transaction.GetKey())
}

// Encode is a part of blockdb.Record interface implementation.
func (tr *txnRecord) Encode(writer io.Writer) error {
	_, err := datastore.ToMsgpack(tr.Transaction).WriteTo(writer)
	return err
}

// Decode is a part of blockdb.Record interface implementation.
func (tr *txnRecord) Decode(reader io.Reader) error {
	return datastore.FromMsgpack(reader, tr.Transaction)
}

type txnRecordProvider struct {
	txnMetadataProvider datastore.EntityMetadata
}

func (trp *txnRecordProvider) NewRecord() blockdb.Record {
	r := &txnRecord{}
	r.Transaction = trp.txnMetadataProvider.Instance().(*transaction.Transaction)
	return r
}

func (bdbs *BlockDBStore) Write(b *block.Block) error {
	db, err := blockdb.NewBlockDB(bdbs.getFileWithoutExtension(b.Hash, b.Round), 64, bdbs.compress)
	if err != nil {
		return err
	}
	var headerBlock = b // maybe use b.Clone()
	headerBlock.Txns = nil
	bh := &blockHeader{Block: headerBlock}
	db.SetDBHeader(bh)
	err = db.Create()
	if err != nil {
		return err
	}
	for _, txn := range b.Txns {
		tr := &txnRecord{Transaction: txn}
		err = db.WriteData(tr)
		if err != nil {
			db.Close()
			return err
		}
	}
	return db.Save()
}

// ReadWithBlockSummary - implement interface
func (bdbs *BlockDBStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	db, err := blockdb.NewBlockDB(bdbs.getFileWithoutExtension(bs.Hash, bs.Round), 64, bdbs.compress)
	if err != nil {
		return nil, err
	}

	block := bdbs.blockMetadataProvider.Instance().(*block.Block)
	bh := &blockHeader{Block: block}
	db.SetDBHeader(bh)
	err = db.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	handler := func(ctx context.Context, record blockdb.Record) error {
		txn := record.(*txnRecord)
		block.Txns = append(block.Txns, txn.Transaction)
		return nil
	}
	trp := &txnRecordProvider{txnMetadataProvider: bdbs.txnMetadataProvider}
	err = db.Iterate(context.Background(), handler, trp)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// DeleteBlock - implement interface
func (bdbs *BlockDBStore) DeleteBlock(b *block.Block) error {
	db, err := blockdb.NewBlockDB(bdbs.getFileWithoutExtension(b.Hash, b.Round), 64, false)
	if err != nil {
		return err
	}
	return db.Delete()
}

func (bdbs *BlockDBStore) UploadToCloud(hash string, round int64) error {
	return common.NewError("interface_not_implemented", "BlockDBStore cannote provide this interface")
}

func (bdbs *BlockDBStore) DownloadFromCloud(hash string, round int64) error {
	return common.NewError("interface_not_implemented", "BlockDBStore cannote provide this interface")
}

func (bdbs *BlockDBStore) CloudObjectExists(hash string) bool {
	return false
}
