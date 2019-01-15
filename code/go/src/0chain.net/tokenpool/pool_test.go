package tokenpool

import (
	"testing"

	"0chain.net/transaction"
)

func TestDigPool(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 8675309
	p := &ZcnPool{}
	transfer, resp, err := p.DigPool("dig_pool", &txn)
	t.Logf("pool: %v; transfer: %v; error: %v; resp: %v\n", p, transfer, err, resp)
	if p.GetBalance() != 8675309 {
		t.Errorf("Pool wasn't dug, balance %v", p.GetBalance())
	}
	t.Logf("pool: %v\n", p)
}

func TestFillPool(t *testing.T) {
	txn := transaction.Transaction{}
	p := &ZcnPool{}
	p.DigPool("fill_pool", &txn)
	t.Logf("pool: %v\n", p)
	txn.Value = 23
	transfer, resp, err := p.FillPool(&txn)
	t.Logf("pool: %v; transfer: %v; error: %v; resp: %v\n", p, transfer, err, resp)
	if p.GetBalance() != 23 {
		t.Error("Pool wasn't filled")
	}
	t.Logf("pool: %v\n", p)
}

func TestEmptyPool(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 32
	p := &ZcnPool{}
	p.DigPool("empty_pool", &txn)
	t.Logf("pool: %v\n", p)
	transfer, resp, err := p.EmptyPool("from_client", "to_client")
	t.Logf("pool: %v; transfer: %v; error: %v; resp: %v\n", p, transfer, err, resp)
	if transfer.Amount != 32 || p.GetBalance() != 0 {
		t.Error("Pool wasn't emptyed properly")
	}
}

func TestDrainPoolWithinBalance(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 33
	p := &ZcnPool{}
	p.DigPool("drain_pool_within_balance", &txn)
	t.Logf("pool: %v\n", p)
	transfer, resp, err := p.DrainPool("from_client", "to_client", 10)
	t.Logf("pool: %v; transfer: %v; error: %v; resp: %v\n", p, transfer, err, resp)
	if transfer.Amount != 10 || p.GetBalance() != 23 || err != nil {
		t.Error("Pool wasn't drained properly")
	}
}

func TestDrainPoolExceedBalance(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 31
	p := &ZcnPool{}
	p.DigPool("drain_pool_exceed_balance", &txn)
	t.Logf("pool: %v\n", p)
	transfer, resp, err := p.DrainPool("from_client", "to_client", 32)
	t.Logf("pool: %v; transfer: %v; error: %v; response: %v\n", p, transfer, err, resp)
	if err == nil || transfer != nil || p.GetBalance() != 31 {
		t.Error("Pool wasn't drained properly")
	}
}

func TestDrainPoolToEmpty(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 37
	p := &ZcnPool{}
	p.DigPool("drain_pool_equals_balance", &txn)
	t.Logf("pool: %v\n", p)
	transfer, resp, err := p.DrainPool("from_client", "to_client", 37)
	t.Logf("pool: %v; transfer: %v; error: %v; response: %v\n", p, transfer, err, resp)
	if transfer.Amount != 37 || p.GetBalance() != 0 || err != nil {
		t.Error("Pool wasn't drained properly")
	}
}

func TestSimpleTransferTo(t *testing.T) {
	txn := transaction.Transaction{}
	p0, p1 := &ZcnPool{}, &ZcnPool{}
	p0.DigPool("pool_0", &txn)
	var err error
	txn.Value = 7
	p1.DigPool("pool_1", &txn)
	t.Logf("pool_0: %v\npool_1: %v\n", p0, p1)
	resp, _ := p1.TransferTo(p0, 1)
	t.Logf("results: %v\n", resp)
	resp, _ = p1.TransferTo(p0, 2)
	t.Logf("results: %v\n", resp)
	resp, _ = p1.TransferTo(p0, 3)
	t.Logf("results: %v\n", resp)
	resp, err = p1.TransferTo(p0, 1)
	t.Logf("results: %v\n", resp)
	if p0.GetBalance() != 7 || p1.GetBalance() != 0 || err != nil {
		t.Error("Pool balance wasn't transfered properly")
	}
}

func TestTransferToAmountExceedsBalance(t *testing.T) {
	txn := transaction.Transaction{}
	p0, p1 := &ZcnPool{}, &ZcnPool{}
	p0.DigPool("pool_0", &txn)
	p1.DigPool("pool_1", &txn)
	_, err := p0.TransferTo(p1, 1948)
	t.Logf("pool_0: %v\npool_1: %v\nerror: %v\n", p0, p1, err)
	if err == nil {
		t.Error("Pool balance wasn't transfered properly")
	}
}

func TestTransferBackAndForth(t *testing.T) {
	txn := transaction.Transaction{}
	p0, p1, p2 := &ZcnPool{}, &ZcnPool{}, &ZcnPool{}
	p0.DigPool("pool_0", &txn)
	txn.Value = 7
	p1.DigPool("pool_1", &txn)
	txn.Value = 9
	p2.DigPool("pool_2", &txn)
	resp, err := p1.TransferTo(p0, 1)
	if err != nil || p0.GetBalance() != 1 || p1.GetBalance() != 6 {
		t.Error("Pool balance wasn't transfered properly")
	} else {
		t.Logf("results: %v\n", resp)
	}
	resp, err = p1.TransferTo(p2, 2)
	if err != nil || p1.GetBalance() != 4 || p2.GetBalance() != 11 {
		t.Error("Pool balance wasn't transfered properly")
	} else {
		t.Logf("results: %v\n", resp)
	}
	resp, err = p2.TransferTo(p0, 8)
	if err != nil || p0.GetBalance() != 9 || p2.GetBalance() != 3 {
		t.Error("Pool balance wasn't transfered properly")
	} else {
		t.Logf("results: %v\n", resp)
	}
	resp, err = p2.TransferTo(p1, 2)
	if err != nil || p1.GetBalance() != 6 || p2.GetBalance() != 1 {
		t.Error("Pool balance wasn't transfered properly")
	} else {
		t.Logf("results: %v\n", resp)
	}
	resp, err = p0.TransferTo(p1, 1)
	if err != nil || p0.GetBalance() != 8 || p1.GetBalance() != 7 {
		t.Error("Pool balance wasn't transfered properly")
	} else {
		t.Logf("results: %v\n", resp)
	}
	resp, err = p0.TransferTo(p2, 8)
	if err != nil || p0.GetBalance() != 0 || p2.GetBalance() != 9 {
		t.Error("Pool balance wasn't transfered properly")
	}
}
