package blockstore

import (
	"context"
	"io"

	"0chain.net/block"
	"0chain.net/blockdb"
	"0chain.net/datastore"
	"0chain.net/transaction"
)

//BlockDBStore is a block store backed by blockdb
type BlockDBStore struct {
	*FSBlockStore
	txnMetadataProvider datastore.EntityMetadata
	compress            bool
}

//NewBlockDBStore - create a new blockdb store
func NewBlockDBStore(rootDir string) BlockStore {
	store := &BlockDBStore{}
	store.compress = true
	store.FSBlockStore = NewFSBlockStore(rootDir)
	store.txnMetadataProvider = datastore.GetEntityMetadata("txn")
	return store
}

type blockHeader struct {
	*block.Block
}

//Encode - implement interface
func (bh *blockHeader) Encode(writer io.Writer) error {
	_, err := datastore.ToMsgpack(bh.Block).WriteTo(writer)
	return err
}

//Decode - implement interface
func (bh *blockHeader) Decode(reader io.Reader) error {
	return datastore.FromMsgpack(reader, bh.Block)
}

type txnRecord struct {
	*transaction.Transaction
}

//GetKey - implement interface
func (tr *txnRecord) GetKey() blockdb.Key {
	return blockdb.Key(tr.Transaction.GetKey())
}

//Encode - implement interface
func (tr *txnRecord) Encode(writer io.Writer) error {
	_, err := datastore.ToMsgpack(tr.Transaction).WriteTo(writer)
	return err
}

//Decode - implement interface
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
	var headerBlock = *b
	headerBlock.Txns = nil
	bh := &blockHeader{Block: &headerBlock}
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

//ReadWithBlockSummary - implement interface
func (bdbs *BlockDBStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	db, err := blockdb.NewBlockDB(bdbs.getFileWithoutExtension(bs.Hash, bs.Round), 64, bdbs.compress)
	block := bdbs.blockMetadataProvider.Instance().(*block.Block)
	bh := &blockHeader{Block: block}
	db.SetDBHeader(bh)
	db.Open()
	defer db.Close()
	handler := func(ctx context.Context, record blockdb.Record) error {
		txn, _ := record.(*txnRecord)
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

//DeleteBlock - implement interface
func (bdbs *BlockDBStore) DeleteBlock(b *block.Block) error {
	db, err := blockdb.NewBlockDB(bdbs.getFileWithoutExtension(b.Hash, b.Round), 64, false)
	if err != nil {
		return err
	}
	return db.Delete()
}
