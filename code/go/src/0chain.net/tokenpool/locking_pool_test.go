package tokenpool

import (
	"encoding/json"
	"testing"
	"time"

	"0chain.net/common"
	"0chain.net/transaction"
)

const (
	LOCKUPTIME90DAYS = time.Duration(time.Second * 10)
	C0               = "client_0"
	C1               = "client_1"
)

type tokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
}

func (tl tokenLock) IsLocked(txn *transaction.Transaction) bool {
	return common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)) < tl.Duration
}

func (tl tokenLock) LockStats(txn *transaction.Transaction) []byte {
	ts := &tokenStat{Locked: tl.IsLocked(txn)}
	return ts.Encode()
}

type tokenStat struct {
	Locked bool `json:"is_locked"`
}

func (ts *tokenStat) Encode() []byte {
	buff, _ := json.Marshal(ts)
	return buff
}

func (ts *tokenStat) Decode(input []byte) error {
	err := json.Unmarshal(input, ts)
	return err
}

func TestTransferToLockPool(t *testing.T) {
	txn := &transaction.Transaction{}
	txn.ClientID = C0
	txn.Value = 10
	txn.CreationDate = common.Now()
	p0 := &ZcnLockingPool{}
	p0.TokenLockInterface = &tokenLock{Duration: LOCKUPTIME90DAYS, StartTime: common.Now()}
	p0.DigPool(C0, txn)

	p1 := &ZcnPool{}
	txn.Value = 2
	txn.ClientID = C1
	txn.CreationDate = common.Now()
	p1.DigPool(C1, txn)

	_, _, err := p0.TransferTo(p1, 9, txn)
	if err == nil {
		t.Errorf("transfer happened before lock expired\n\tstart time: %v\n\ttxn time: %v\n", p0.IsLocked(txn), txn.CreationDate)
	}

	time.Sleep(LOCKUPTIME90DAYS)
	txn.CreationDate = common.Now()
	_, _, err = p0.TransferTo(p1, 9, txn)
	if err != nil {
		t.Errorf("an error occoured %v\n", err.Error())
	} else if p1.Balance != 11 {
		t.Errorf("pool 1 has wrong balance: %v\ntransaction time: %v\n", p1, common.ToTime(txn.CreationDate))
	}
}
