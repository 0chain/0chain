package block

import (
	"context"
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

func (m *MagicBlockData) Delete(ctx context.Context) error {
	return m.GetEntityMetadata().GetStore().Delete(ctx, m)
}

func NewMagicBlockData(mb *MagicBlock) *MagicBlockData {
	mbData := datastore.GetEntityMetadata("magicblockdata").Instance().(*MagicBlockData)
	mbData.ID = strconv.FormatInt(mb.MagicBlockNumber, 10)
	mbData.MagicBlock = mb
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
	defer ememorystore.Close(dctx)

	if err = mbd.Read(dctx, mbd.GetKey()); err != nil {
		return
	}
	mb = mbd.MagicBlock
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
	defer ememorystore.Close(rctx)

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
