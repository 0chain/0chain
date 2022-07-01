package event

import (
	"errors"
	"sync"

	"0chain.net/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// swagger:model Block
type Snapshot struct {
	gorm.Model
	Round           int64  `gorm:"primaryKey;autoIncrement:false" json:"round"`
	BlockHash       string `json:"block_hash"`
	MintTotalAmount int64  `json:"mint_total_amount"`
}

var snapshotLock sync.Mutex

func (edb *EventDb) GetRoundsMintTotal(from, to int64) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&Snapshot{}).Where("round between ? and ?", from, to).Select("sum(mint_total_amount)").Scan(&total).Error
}

func (edb *EventDb) addOrUpdateTotalMint(mint Mint) error {
	snapshotLock.Lock()
	defer snapshotLock.Unlock()

	snapshot, err := edb.getRoundSnapshot(mint.Round)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			snapshot = &Snapshot{
				Round:           mint.Round,
				MintTotalAmount: mint.Amount,
				BlockHash:       mint.BlockHash,
			}
			return edb.addSnapshot(snapshot)
		} else {
			return err
		}
	}
	snapshot.MintTotalAmount = mint.Amount
	logging.Logger.Debug("snapshot found: ", zap.Any("snapshot", snapshot))
	edb.Store.Get().Save(&snapshot)
	return nil
}

func (edb *EventDb) getRoundSnapshot(round int64) (*Snapshot, error) {
	snapshot := &Snapshot{}
	res := edb.Store.Get().Table("snapshots").Where("round = ?", round).First(&snapshot)
	return snapshot, res.Error
}

func (edb *EventDb) addSnapshot(snapshot *Snapshot) error {
	result := edb.Store.Get().Create(snapshot)
	return result.Error
}
