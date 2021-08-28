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
	BUCKET              = "bmr"
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
		_, err := t.CreateBucketIfNotExists([]byte(BUCKET)) //block meta record
		return err
	}); err != nil {
		return err
	}
	return err
}

// It would be better to save this meta record in hdd/s3 as per tiering config so that upon hot tiered disk fails it still can
// be reconstructed. Writing to multiple disks but makes block writing as a whole a slow process.
func (bmr *BlockMetaRecord) AddOrUpdate() (err error) {
	err = db.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BUCKET))
		if bkt == nil {
			return errors.New("Bucket for Block meta recording not found")
		}
		data, err := json.Marshal(bmr)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(bmr.Hash), data)
	})

	return
}

func GetBlockMetaRecord(hash string) (*BlockMetaRecord, error) {
	var data []byte
	var bmr BlockMetaRecord
	err := db.View(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BUCKET))
		if bkt == nil {
			return errors.New("Bucket for Block meta recording not found")
		}
		data = bkt.Get([]byte(hash))
		if data == nil {
			return fmt.Errorf("Block meta record for %v not found.", hash)
		}
		return json.Unmarshal(data, &bmr)
	})
	if err != nil {
		return nil, err
	}
	bmr.Hash = hash
	return &bmr, nil
}

func GetAllRecord(ch chan<- BlockMetaRecord) {
	err := db.View(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BUCKET))
		if bkt == nil {
			return errors.New("Bucket for Block meta recording not found")
		}
		err := bkt.ForEach(func(k, v []byte) error {
			bmr := BlockMetaRecord{Hash: string(k)}
			err := json.Unmarshal(v, &bmr)
			if err != nil {
				return err
			}
			ch <- bmr
			return nil
		})

		return err
	})
	if err != nil {
		close(ch)
	}
}
