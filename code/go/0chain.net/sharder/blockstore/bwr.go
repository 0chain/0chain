package blockstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"0chain.net/core/ememorystore"
	"0chain.net/core/viper"
	"github.com/0chain/gorocksdb"
)

type WhichTier uint8

const (
	DiskTier        WhichTier = 2 // Block is in disk only
	ColdTier        WhichTier = 4 // Block is in cold storage only
	DiskAndColdTier WhichTier = 8 // Block is in both disk and cold storages
)

const (
	BWRCF = "bwr" // bwr column family
	UBRCF = "ubr" // ubr column family
	BMR   = "bmr" // block meta record
)

var bmrDB *gorocksdb.DB
var bwrHandle *gorocksdb.ColumnFamilyHandle
var ubrHandle *gorocksdb.ColumnFamilyHandle

type blockWhereRecord struct {
	Hash    string    `json:"-"`
	Tiering WhichTier `json:"tr"`
	//For disk volume it is simple unix path. For cold storage it is "storageUrl:bucketName"
	BlockPath string `json:"vp,omitempty"`
	ColdPath  string `json:"cp,omitempty"`
}

func (bwr *blockWhereRecord) addOrUpdate() error {
	data, err := json.Marshal(bwr)
	if err != nil {
		return err
	}
	wo := gorocksdb.NewDefaultWriteOptions()
	err = bmrDB.PutCF(gorocksdb.NewDefaultWriteOptions(), bwrHandle, []byte(bwr.Hash), data)
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
	return bwr, nil
}

/**********************************Unmoved block record***************************************/
type unmovedBlockRecord struct {
	Hash string `json:"h"`
	// CreateAt duration passed from epoch date
	CreatedAt time.Duration `json:"c"`
}

func (ubr *unmovedBlockRecord) Add() error {
	k := strconv.FormatInt(int64(ubr.CreatedAt), 64)
	wo := gorocksdb.NewDefaultWriteOptions()
	err := bmrDB.PutCF(gorocksdb.NewDefaultWriteOptions(), ubrHandle, []byte(k), []byte(ubr.Hash))
	wo.Destroy()
	return err
}

func (ubr *unmovedBlockRecord) Delete() error {
	k := strconv.FormatInt(int64(ubr.CreatedAt), 64)
	wo := gorocksdb.NewDefaultWriteOptions()
	err := bmrDB.DeleteCF(wo, ubrHandle, []byte(k))
	wo.Destroy()
	return err
}

// getUnmovedBlockRecords will return a channel where it will pass
// all the unmoved blocks
func getUnmovedBlockRecords(maxPrefix []byte) <-chan *unmovedBlockRecord {
	ch := make(chan *unmovedBlockRecord)

	go func() {
		defer close(ch)

		ro := gorocksdb.NewDefaultReadOptions()
		defer ro.Destroy()

		it := bmrDB.NewIteratorCF(ro, ubrHandle)
		defer it.Close()
		for it.SeekToFirst(); it.Valid() && bytes.Compare(it.Key().Data(), maxPrefix) != 1; it.Next() {
			keyS := it.Key()
			valueS := it.Value()

			createdAt, _ := strconv.ParseInt(string(keyS.Data()), 10, 64)
			ch <- &unmovedBlockRecord{
				Hash:      string(valueS.Data()),
				CreatedAt: time.Duration(createdAt),
			}
			keyS.Free()
			valueS.Free()
		}
	}()
	return ch
}

func initBWR(viper *viper.Viper, mode, workDir string) {
	if viper == nil {
		panic("bwr config not provided")
	}

	dbPath := filepath.Join(workDir, "data/rocksdb", BMR)
	cacheSize, err := getUint64ValueFromYamlConfig(viper.GetString("cache_size"))
	if err != nil {
		panic(err)
	}

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
		bmrDB, cfHs, err = ememorystore.OpenDB(dbPath, cfs, cfsOpts, cacheSize, false)
		bwrHandle = cfHs[1]
		ubrHandle = cfHs[2]

	default:
		err := os.RemoveAll(dbPath)
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}

		err = os.MkdirAll(dbPath, 0777)
		if err != nil {
			panic(err)
		}

		bmrDB, _, err = ememorystore.OpenDB(dbPath, nil, nil, cacheSize, true)
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
