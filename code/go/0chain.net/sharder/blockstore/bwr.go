package blockstore

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"0chain.net/core/common"
	"0chain.net/core/ememorystore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/gorocksdb"
)

type WhichTier uint8

const (
	DiskTier        WhichTier = 2 // Block is in disk only
	ColdTier        WhichTier = 4 // Block is in cold storage only
	DiskAndColdTier WhichTier = 8 // Block is in both disk and cold storages
)

const (
	// bwr(block where record) column family
	BWRCF = "bwr"
	// ubr(unmoved block record) column family
	UBRCF = "ubr"
	// block meta record
	BMR = "bmr"
)

var bmrDB *gorocksdb.DB
var bwrHandle *gorocksdb.ColumnFamilyHandle // column family handle for block where record
var ubrHandle *gorocksdb.ColumnFamilyHandle // column family handle for unmoved block record

type blockWhereRecord struct {
	Hash    string    `json:"-"`
	Tiering WhichTier `json:"tr"`
	//For disk volume it is simple unix path. For cold storage it is "storageUrl:bucketName"
	BlockPath string `json:"bp,omitempty"`
	ColdPath  string `json:"cp,omitempty"`
}

func (bwr *blockWhereRecord) save() error {
	data, err := json.Marshal(bwr)
	if err != nil {
		return err
	}
	wo := gorocksdb.NewDefaultWriteOptions()
	err = bmrDB.PutCF(wo, bwrHandle, []byte(bwr.Hash), data)
	wo.Destroy()
	return err
}

func getBWR(hash string) (*blockWhereRecord, error) {
	ro := gorocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	dataS, err := bmrDB.GetCF(ro, bwrHandle, []byte(hash))
	if err != nil {
		return nil, err
	}
	defer dataS.Free()

	data := dataS.Data()
	bwr := &blockWhereRecord{}
	err = json.Unmarshal(data, bwr)
	if err != nil {
		return nil, err
	}
	bwr.Hash = hash
	return bwr, nil
}

/**********************************Unmoved block record***************************************/
type unmovedBlockRecord struct {
	Hash string `json:"h"`
	// CreateAt duration passed from epoch date
	CreatedAt common.Timestamp `json:"c"`
}

func (ubr *unmovedBlockRecord) Add() error {
	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, binary.LittleEndian, ubr.CreatedAt); err != nil {
		return err
	}
	key := buf.Bytes()
	wo := gorocksdb.NewDefaultWriteOptions()
	err := bmrDB.PutCF(gorocksdb.NewDefaultWriteOptions(), ubrHandle, key, []byte(ubr.Hash))
	wo.Destroy()
	return err
}

func (ubr *unmovedBlockRecord) Delete() error {
	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, binary.LittleEndian, ubr.CreatedAt); err != nil {
		return err
	}
	key := buf.Bytes()
	wo := gorocksdb.NewDefaultWriteOptions()
	err := bmrDB.DeleteCF(wo, ubrHandle, key)
	wo.Destroy()
	return err
}

// getUnmovedBlockRecords will return a channel where it will pass
// all the unmoved blocks that is older than the block movement time interval
func getUnmovedBlockRecords(uptoDateU common.Timestamp) <-chan *unmovedBlockRecord {
	ch := make(chan *unmovedBlockRecord)
	go func() {
		defer close(ch)

		ro := gorocksdb.NewDefaultReadOptions()
		defer ro.Destroy()

		it := bmrDB.NewIteratorCF(ro, ubrHandle)
		defer it.Close()

		var createdAt common.Timestamp
		for it.SeekToFirst(); it.Valid() && func(it *gorocksdb.Iterator) bool {
			r := bytes.NewReader(it.Key().Data())
			err := binary.Read(r, binary.LittleEndian, &createdAt)
			if err != nil {
				logging.Logger.Error(err.Error())
				return false
			}
			return uptoDateU > createdAt
		}(it); it.Next() { // Key should not be greater than maxPrefix

			ch <- &unmovedBlockRecord{
				Hash:      string(it.Value().Data()),
				CreatedAt: createdAt,
			}
			it.Key().Free()
			it.Value().Free()
		}
	}()
	return ch
}

func initBlockWhereRecord(cacheSize uint64, mode, workDir, dirname string) {
	dbPath := filepath.Join(workDir, dirname, "data/rocksdb", BMR)
	cfs := []string{"default", BWRCF, UBRCF}
	bwrOpt := gorocksdb.NewDefaultOptions()
	bwrOpt.OptimizeForPointLookup(cacheSize)
	bwrOpt.SetAllowConcurrentMemtableWrites(false)

	ubrOpt := gorocksdb.NewDefaultOptions()

	cfsOpts := []*gorocksdb.Options{gorocksdb.NewDefaultOptions(), bwrOpt, ubrOpt}
	switch mode {
	case "restart":
		_, err := os.Stat(dbPath)
		if err != nil {
			panic(fmt.Errorf("mode is %s, error: %s", mode, err.Error()))
		}

		var cfHs gorocksdb.ColumnFamilyHandles
		bmrDB, cfHs, err = ememorystore.OpenDBWithColumnFamilies(dbPath, cfs, cfsOpts, cacheSize, false)
		if err != nil {
			panic(fmt.Errorf("error while opening rocksdb. Path: %s, error: %s", dbPath, err.Error()))
		}
		bwrHandle = cfHs[1]
		ubrHandle = cfHs[2]

	default:
		err := os.RemoveAll(dbPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}

		err = os.MkdirAll(dbPath, 0777)
		if err != nil {
			panic(err)
		}

		bmrDB, _, err = ememorystore.OpenDBWithColumnFamilies(dbPath, nil, nil, cacheSize, true)
		if err != nil {
			panic(err)
		}

		bwrHandle, err = bmrDB.CreateColumnFamily(bwrOpt, BWRCF)
		if err != nil {
			panic(err)
		}
		ubrHandle, err = bmrDB.CreateColumnFamily(ubrOpt, UBRCF)
		if err != nil {
			panic(err)
		}
	}

	bwrOpt.Destroy()
	ubrOpt.Destroy()
}
