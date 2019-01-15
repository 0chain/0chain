package tokenpool

import (
	"testing"
	"time"

	"0chain.net/transaction"
)

const (
	LOCKUPTIME90DAYS = time.Duration(time.Second * 10)
	C0               = "client_0"
	C1               = "client_1"
)

func TestTransferToLockPool(t *testing.T) {
	txn := transaction.Transaction{}
	txn.ClientID = C0
	txn.Value = 10
	p0 := &ZcnLockingPool{}
	p0.Duration = LOCKUPTIME90DAYS
	p0.StartTime = time.Now()
	transfer, resp, err := p0.DigPool(C0, &txn)
	t.Logf("pool: %v\ntransfer: %v\nerror: %v\nresp: %v\n", p0, transfer, err, resp)
	if p0.GetBalance() != 10 {
		t.Errorf("Pool wasn't dug, balance %v", p0.GetBalance())
	}
	p1 := &ZcnLockingPool{}
	p1.Duration = LOCKUPTIME90DAYS
	p1.StartTime = time.Now()
	txn.Value = 2
	txn.ClientID = C1
	p1.DigPool(C1, &txn)
	t.Logf("pool: %v\n", p0)
	t.Logf("pool: %v\n", p1)
	str, err := p0.TransferTo(p1, 9)
	if err == nil {
		t.Logf("str: %v\n", str)
	} else {
		t.Logf("err: %v\n", err.Error())
	}
	t.Logf("pool: %v\n", p0)
	t.Logf("pool: %v\n", p1)

	time.Sleep(LOCKUPTIME90DAYS)
	str, err = p0.TransferTo(p1, 9)
	if err == nil {
		t.Logf("str: %v\n", str)
	} else {
		t.Logf("err: %v\n", err.Error())
	}
	t.Logf("pool: %v\n", p0)
	t.Logf("pool: %v\n", p1)
}
