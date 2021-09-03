package interestpoolsc

import (
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

type tokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
	Owner     datastore.Key    `json:"owner"`
}

func (tl tokenLock) IsLocked(entity interface{}) bool {
	tm, ok := entity.(time.Time)
	if ok {
		return tm.Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	tm, ok := entity.(time.Time)
	if ok {
		p := &poolStat{StartTime: tl.StartTime, Duartion: tl.Duration, TimeLeft: (tl.Duration - tm.Sub(common.ToTime(tl.StartTime))), Locked: tl.IsLocked(tm)}
		return p.encode()
	}
	return nil
}
