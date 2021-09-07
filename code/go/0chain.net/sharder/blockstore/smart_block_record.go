package blockstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

type Tier uint8

const (
	BMRBucket           = "bmr"
	UnmovedBlocks       = "ubs"
	HotTier        Tier = iota //Hot tier only
	WarmTier                   //Warm tier only
	ColdTier                   //Cold tier only
	HotAndWarmTier             //Hot and warm tier
	HotAndColdTier             //Hot and cold tier
)

var (
	db *bbolt.DB
)

type BlockMetaRecord struct {
	Hash              string    `json:"-"`
	LastAccessTime    time.Time `json:"lrt"`
	LatestAccessCount uint64    `json:"lac"` //This will reset to 0 when moving from hot tier.
	Tiering           int       `json:"tr"`
	VolumePath        string    `json:"vp"`
}

func NewBlockMetaStore(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0644)
	if err != nil {
		return err
	}
	db, err = bbolt.Open(path, 0644, nil)
	if err := db.Update(func(t *bbolt.Tx) error {
		_, err := t.CreateBucketIfNotExists([]byte(BMRBucket)) //block meta record
		return err
	}); err != nil {
		return err
	}
	return err
}

func (bmr *BlockMetaRecord) AddOrUpdate() (err error) {
	value, err := json.Marshal(bmr)
	if err != nil {
		return err
	}
	key := []byte(bmr.Hash)
	err = db.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BMRBucket))
		if bkt == nil {
			return errors.New("Bucket for Block meta recording not found")
		}
		return bkt.Put(key, value)
	})

	return
}

func GetBlockMetaRecord(hash string) (*BlockMetaRecord, error) {
	var data []byte
	var bmr BlockMetaRecord
	key := []byte(hash)
	err := db.View(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BMRBucket))
		if bkt == nil {
			return errors.New("Bucket for Block meta recording not found")
		}
		bmrData := bkt.Get(key)
		if bmrData == nil {
			return fmt.Errorf("Block meta record for %v not found.", hash)
		}
		data = make([]byte, len(bmrData))
		copy(data, bmrData)
		return nil
	})

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &bmr)
	if err != nil {
		return nil, err
	}
	bmr.Hash = hash
	return &bmr, nil
}

type ColdBlock struct {
	HashKey   string
	CreatedAt time.Time
}

//Add block hash to db with current time so upon some duration passed it can be moved to cold tier if cold tiering
//is enabled
func AddToMoveBlocks(hashKey string) error {
	nowByte, _ := time.Now().MarshalText()
	hashKeyByte := []byte(hashKey)
	return db.Update(func(t *bbolt.Tx) error {
		bkt, err := t.CreateBucketIfNotExists([]byte(UnmovedBlocks))
		if err != nil {
			return err
		}
		return bkt.Put(hashKeyByte, nowByte)
	})
}

//Remove block hash from db as it is moved to cold tier
func RemoveMovedBlocks(hashKey string) error {
	hashKeyByte := []byte(hashKey)
	return db.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(UnmovedBlocks))
		if bkt == nil {
			return nil
		}
		return bkt.Delete(hashKeyByte)
	})
}

//Get all unmoved block hashes. Since sharder can choose to transfer blocks to cold tier in any duration i.e. one day, one week;
//monthly, etc. so it is iterated based on the prefix key. There is chance of block being missed to be transferred to cold
//tier because of the iteration method choosed below but it will be moved within few tiering.
//Reading all possibly million keys in single transaction takes some time which will affect block storage in SSD as well.
func GetUnmovedBlocks(ch chan<- *ColdBlock, prefix string) error {
	var hashKey []byte
	var timeByte []byte
	err := db.View(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(UnmovedBlocks))
		if bkt == nil {
			return fmt.Errorf("No bucket")
		}
		cursor := bkt.Cursor()
		k, _ := cursor.Seek([]byte(prefix))
		if k == nil {
			return fmt.Errorf("End of keys")
		}
		k, v := cursor.Next()
		hashKey = make([]byte, len(k))
		timeByte = make([]byte, len(v))
		copy(hashKey, k)
		copy(timeByte, v)
		return nil
	})

	if err != nil {
		return err
	}
	coldBlock := ColdBlock{
		HashKey: string(hashKey),
	}
	coldBlock.CreatedAt.UnmarshalText(timeByte)

	ch <- &coldBlock
	return nil
}
