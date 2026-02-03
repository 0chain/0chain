package block

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

type MagicBlockData struct {
	datastore.IDField
	*MagicBlock
	Data []byte
}

var magicBlockMetadata *datastore.EntityMetadataImpl

func (m *MagicBlockData) GetEntityMetadata() datastore.EntityMetadata {
	return magicBlockMetadata
}

func MagicBlockDataProvider() datastore.Entity {
	return &MagicBlockData{}
}

func SetupMagicBlockData(store datastore.Store) {
	magicBlockMetadata = datastore.MetadataProvider()
	magicBlockMetadata.Name = "magicblockdata"
	magicBlockMetadata.DB = "magicblockdatadb"
	magicBlockMetadata.Store = store
	magicBlockMetadata.Provider = MagicBlockDataProvider
	datastore.RegisterEntityMetadata("magicblockdata", magicBlockMetadata)
}

func SetupMagicBlockDataDB(workdir string) {
	db, err := ememorystore.CreateDB(filepath.Join(workdir, "data/rocksdb/mb"))
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("magicblockdatadb", db)
}

func (m *MagicBlockData) Read(ctx context.Context, key string) error {
	return m.GetEntityMetadata().GetStore().Read(ctx, key, m)
}

func (m *MagicBlockData) Write(ctx context.Context) error {
	return m.GetEntityMetadata().GetStore().Write(ctx, m)
}

// SaveMagicBlock saves a magic block to RocksDB with proper transaction handling.
// This function creates a new transaction, writes the data, commits, and closes.
// Use this instead of MagicBlockData.Write() when you need writes to persist.
func SaveMagicBlock(ctx context.Context, mb *MagicBlock) error {
	mbData := NewMagicBlockData(mb)
	emd := mbData.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, emd)

	// Write the data
	if err := mbData.Write(dctx); err != nil {
		ememorystore.Close(dctx, emd)
		return fmt.Errorf("failed to write magic block %d: %v", mb.MagicBlockNumber, err)
	}

	// Commit the transaction to persist data
	conn := ememorystore.GetEntityCon(dctx, emd)
	if err := conn.Commit(); err != nil {
		ememorystore.Close(dctx, emd)
		return fmt.Errorf("failed to commit magic block %d: %v", mb.MagicBlockNumber, err)
	}

	// Close (will not rollback since we committed)
	ememorystore.Close(dctx, emd)
	return nil
}

func (m *MagicBlockData) Delete(ctx context.Context) error {
	return m.GetEntityMetadata().GetStore().Delete(ctx, m)
}

func NewMagicBlockData(mb *MagicBlock) *MagicBlockData {
	mbData := datastore.GetEntityMetadata("magicblockdata").Instance().(*MagicBlockData)
	mbData.ID = strconv.FormatInt(mb.MagicBlockNumber, 10)

	d, err := mb.MarshalMsg(nil)
	if err != nil {
		logging.Logger.Panic(fmt.Sprintf("[mvc] failed to marshal magic block: %v", err))
	}

	mbData.Data = d
	return mbData
}

func LoadMagicBlock(ctx context.Context, id string) (mb *MagicBlock,
	err error) {

	var mbd = datastore.GetEntity("magicblockdata").(*MagicBlockData)
	mbd.ID = id

	var (
		emd  = mbd.GetEntityMetadata()
		dctx = ememorystore.WithEntityConnection(ctx, emd)
	)
	defer ememorystore.Close(dctx, emd)

	if err = mbd.Read(dctx, mbd.GetKey()); err != nil {
		return
	}

	if len(mbd.Data) == 0 && mbd.MagicBlock != nil {
		mb = mbd.MagicBlock
		return
	}

	var inMB MagicBlock
	if _, err := inMB.UnmarshalMsg(mbd.Data); err != nil {
		logging.Logger.Error("[mvc] failed to unmarshal magic block", zap.Error(err))
		return nil, fmt.Errorf("could not decode magic block: %v", err)
	}

	logging.Logger.Debug("[mvc] load mb", zap.Int64("mb number from data", inMB.MagicBlockNumber))
	mb = &inMB
	return
}

func LoadLatestMB(ctx context.Context, lfbRound, mbNumber int64) (mb *MagicBlock, err error) {
	if mbNumber > 0 {
		mbStr := strconv.FormatInt(mbNumber, 10)
		mb, err = LoadMagicBlock(ctx, mbStr)
		if err != nil {
			logging.Logger.Error("load_latest_mb", zap.Error(err), zap.Int64("mb number", mbNumber))
			return
		}
		logging.Logger.Info("[mvc] find latest MB by magic bock number", zap.Int64("mb number", mbNumber))
		return mb, nil
	}

	var (
		mbemd = datastore.GetEntityMetadata("magicblockdata")
		rctx  = ememorystore.WithEntityConnection(ctx, mbemd)
		conn  = ememorystore.GetEntityCon(rctx, mbemd)
	)
	defer ememorystore.Close(rctx, mbemd)

	iter := conn.Conn.NewIterator(conn.ReadOptions)
	defer iter.Close()
	// the first time the hardfork is happened
	var data = mbemd.Instance().(*MagicBlockData)
	iter.SeekToLast() // from last

	if !iter.Valid() {
		return nil, util.ErrValueNotPresent
	}

	if err = datastore.FromJSON(iter.Value().Data(), data); err != nil {
		return nil, common.NewErrorf("load_latest_mb",
			"decoding error: %v, key: %q", err, string(iter.Key().Data()))
	}

	mb = data.MagicBlock
	logging.Logger.Info("[mvc] seek to the last in MB store", zap.Int64("mb number", mb.MagicBlockNumber))
	return
}

func LoadLatestMBs(ctx context.Context, fromMBNumber int64) (mbs []*MagicBlock) {
	// iterate from fromMBNumber back 5 or till 1,
	var count = 5
	for i := fromMBNumber; i > 0 && count > 0; i-- {
		count--
		mbStr := strconv.FormatInt(i, 10)
		mb, err := LoadMagicBlock(ctx, mbStr)
		if err != nil {
			logging.Logger.Error("load_latest_mb", zap.Error(err), zap.Int64("mb number", i))
			continue
		}
		logging.Logger.Info("[mvc] load latest MB from store", zap.Int64("mb number", mb.MagicBlockNumber))
		mbs = append(mbs, mb)
	}

	return mbs
}
