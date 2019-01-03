package pool

import (
	"testing"

	"0chain.net/transaction"
)

func TestDigPool(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 8675309
	p := DigPool("dig_pool", &txn)
	if p.balance != 8675309 {
		t.Error("Pool wasn't dug")
	}
	t.Logf("pool: %v\n", p)
}

func TestFillPool(t *testing.T) {
	txn := transaction.Transaction{}
	p := DigPool("fill_pool", &txn)
	t.Logf("pool: %v\n", p)
	txn.Value = 23
	p.FillPool(&txn)
	if p.balance != 23 {
		t.Error("Pool wasn't filled")
	}
	t.Logf("pool: %v\n", p)
}

func TestEmptyPool(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 32
	p := DigPool("empty_pool", &txn)
	t.Logf("pool: %v\n", p)
	transfer := p.EmptyPool("from_client", "to_client")
	t.Logf("pool: %v; transfer: %v\n", p, transfer)
	if transfer.Amount != 32 || p.balance != 0 {
		t.Error("Pool wasn't emptyed properly")
	}
}

func TestDrainPoolWithinBalance(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 33
	p := DigPool("drain_pool_within_balance", &txn)
	t.Logf("pool: %v\n", p)
	transfer, err := p.DrainPool("from_client", "to_client", 10)
	t.Logf("pool: %v; transfer: %v; error: %v\n", p, transfer, err)
	if transfer.Amount != 10 || p.balance != 23 || err != nil {
		t.Error("Pool wasn't drained properly")
	}
}

func TestDrainPoolExceedBalance(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 31
	p := DigPool("drain_pool_exceed_balance", &txn)
	t.Logf("pool: %v\n", p)
	transfer, err := p.DrainPool("from_client", "to_client", 32)
	t.Logf("pool: %v; transfer: %v; error: %v\n", p, transfer, err)
	if err == nil || transfer != nil || p.balance != 31 {
		t.Error("Pool wasn't drained properly")
	}
}

func TestDrainPoolToEmpty(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 37
	p := DigPool("drain_pool_equals_balance", &txn)
	t.Logf("pool: %v\n", p)
	transfer, err := p.DrainPool("from_client", "to_client", 37)
	t.Logf("pool: %v; transfer: %v; error: %v\n", p, transfer, err)
	if transfer.Amount != 37 || p.balance != 0 || err != nil {
		t.Error("Pool wasn't drained properly")
	}
}

func TestSimpleTransferTo(t *testing.T) {
	txn := transaction.Transaction{}
	p0 := DigPool("pool_0", &txn)
	txn.Value = 7
	p1 := DigPool("pool_1", &txn)
	t.Logf("pool_0: %v\npool_1: %v\n", p0, p1)
	err := p1.TransferTo(p0, 1)
	t.Logf("pool_0: %v\npool_1: %v\nerror: %v\n", p0, p1, err)
	err = p1.TransferTo(p0, 2)
	t.Logf("pool_0: %v\npool_1: %v\nerror: %v\n", p0, p1, err)
	err = p1.TransferTo(p0, 3)
	t.Logf("pool_0: %v\npool_1: %v\nerror: %v\n", p0, p1, err)
	err = p1.TransferTo(p0, 1)
	t.Logf("pool_0: %v\npool_1: %v\nerror: %v\n", p0, p1, err)
	if p0.balance != 7 || p1.balance != 0 || err != nil {
		t.Error("Pool balance wasn't transfered properly")
	}
}

func TestTransferToAmountExceedsBalance(t *testing.T) {
	txn := transaction.Transaction{}
	p0 := DigPool("pool_0", &txn)
	p1 := DigPool("pool_1", &txn)
	err := p0.TransferTo(p1, 1948)
	t.Logf("pool_0: %v\npool_1: %v\nerror: %v\n", p0, p1, err)
	if err == nil {
		t.Error("Pool balance wasn't transfered properly")
	}
}

func TestTransferBackAndForth(t *testing.T) {
	txn := transaction.Transaction{}
	p0 := DigPool("pool_0", &txn)
	txn.Value = 7
	p1 := DigPool("pool_1", &txn)
	txn.Value = 9
	p2 := DigPool("pool_2", &txn)
	err := p1.TransferTo(p0, 1)
	if err != nil || p0.balance != 1 || p1.balance != 6 {
		t.Error("Pool balance wasn't transfered properly")
	}
	err = p1.TransferTo(p2, 2)
	if err != nil || p1.balance != 4 || p2.balance != 11 {
		t.Error("Pool balance wasn't transfered properly")
	}
	err = p2.TransferTo(p0, 8)
	if err != nil || p0.balance != 9 || p2.balance != 3 {
		t.Error("Pool balance wasn't transfered properly")
	}
	err = p2.TransferTo(p1, 2)
	if err != nil || p1.balance != 6 || p2.balance != 1 {
		t.Error("Pool balance wasn't transfered properly")
	}
	err = p0.TransferTo(p1, 1)
	if err != nil || p0.balance != 8 || p1.balance != 7 {
		t.Error("Pool balance wasn't transfered properly")
	}
	err = p0.TransferTo(p2, 8)
	if err != nil || p0.balance != 0 || p2.balance != 9 {
		t.Error("Pool balance wasn't transfered properly")
	}
}
