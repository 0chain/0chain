package tokenpool

import (
	"testing"

	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/logging"
)

func init() {
	logging.InitLogging("development", "")
}

func TestDigPool(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 8675309
	p := &ZcnPool{}
	transfer, _, _ := p.DigPool("dig_pool", &txn)
	if p.GetBalance() != 8675309 || transfer.Amount != p.GetBalance() {
		t.Errorf("Pool wasn't dug, balance %v", p.GetBalance())
	}
}

func TestFillPool(t *testing.T) {
	txn := transaction.Transaction{}
	p := &ZcnPool{}
	_, _, err := p.DigPool("fill_pool", &txn)
	require.NoError(t, err)
	txn.Value = 23
	transfer, _, _ := p.FillPool(&txn)
	if p.GetBalance() != 23 || transfer.Amount != p.GetBalance() {
		t.Error("Pool wasn't filled")
	}
}

func TestEmptyPool(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 32
	p := &ZcnPool{}
	_, _, err := p.DigPool("empty_pool", &txn)
	require.NoError(t, err)
	transfer, _, _ := p.EmptyPool("from_client", "to_client", &txn)
	if transfer.Amount != 32 || p.GetBalance() != 0 {
		t.Error("Pool wasn't emptyed properly")
	}
}

func TestDrainPoolWithinBalance(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 33
	p := &ZcnPool{}
	_, _, err := p.DigPool("drain_pool_within_balance", &txn)
	require.NoError(t, err)
	transfer, _, _ := p.DrainPool("from_client", "to_client", 10, &txn)
	if transfer.Amount != 10 || p.GetBalance() != 23 {
		t.Error("Pool wasn't drained properly")
	}
}

func TestDrainPoolExceedBalance(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 31
	p := &ZcnPool{}
	_, _, err := p.DigPool("drain_pool_exceed_balance", &txn)
	require.NoError(t, err)
	transfer, _, err := p.DrainPool("from_client", "to_client", 32, &txn)
	if err == nil || transfer != nil || p.GetBalance() != 31 {
		t.Error("Pool wasn't drained properly")
	}
}

func TestDrainPoolToEmpty(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Value = 37
	p := &ZcnPool{}
	_, _, err := p.DigPool("drain_pool_equals_balance", &txn)
	require.NoError(t, err)
	transfer, _, err := p.DrainPool("from_client", "to_client", 37, &txn)
	require.NoError(t, err)
	if transfer.Amount != 37 || p.GetBalance() != 0 {
		t.Error("Pool wasn't drained properly")
	}
}

func TestSimpleTransferTo(t *testing.T) {
	txn := transaction.Transaction{}
	p0, p1 := &ZcnPool{}, &ZcnPool{}
	_, _, err := p0.DigPool("pool_0", &txn)
	require.NoError(t, err)
	txn.Value = 7
	_, _, err = p1.DigPool("pool_1", &txn)
	require.NoError(t, err)
	_, _, err = p1.TransferTo(p0, 1, &txn)
	require.NoError(t, err)
	_, _, err = p1.TransferTo(p0, 2, &txn)
	require.NoError(t, err)
	_, _, err = p1.TransferTo(p0, 3, &txn)
	require.NoError(t, err)
	_, _, err = p1.TransferTo(p0, 1, &txn)
	require.NoError(t, err)
	if p0.GetBalance() != 7 || p1.GetBalance() != 0 {
		t.Error("Pool balance wasn't transfered properly")
	}
}

func TestTransferToAmountExceedsBalance(t *testing.T) {
	txn := transaction.Transaction{}
	p0, p1 := &ZcnPool{}, &ZcnPool{}
	_, _, err := p0.DigPool("pool_0", &txn)
	require.NoError(t, err)
	_, _, err = p1.DigPool("pool_1", &txn)
	require.NoError(t, err)
	_, _, err = p0.TransferTo(p1, 1948, &txn)
	if err == nil {
		t.Error("Pool balance wasn't transfered properly")
	}
}

func TestTransferBackAndForth(t *testing.T) {
	txn := transaction.Transaction{}
	p0, p1, p2 := &ZcnPool{}, &ZcnPool{}, &ZcnPool{}
	_, _, err := p0.DigPool("pool_0", &txn)
	require.NoError(t, err)
	txn.Value = 7
	_, _, err = p1.DigPool("pool_1", &txn)
	require.NoError(t, err)
	txn.Value = 9
	_, _, err = p2.DigPool("pool_2", &txn)
	require.NoError(t, err)
	_, _, err = p1.TransferTo(p0, 1, &txn)
	if err != nil || p0.GetBalance() != 1 || p1.GetBalance() != 6 {
		t.Error("Pool balance wasn't transfered properly")
	}
	_, _, err = p1.TransferTo(p2, 2, &txn)
	if err != nil || p1.GetBalance() != 4 || p2.GetBalance() != 11 {
		t.Error("Pool balance wasn't transfered properly")
	}
	_, _, err = p2.TransferTo(p0, 8, &txn)
	if err != nil || p0.GetBalance() != 9 || p2.GetBalance() != 3 {
		t.Error("Pool balance wasn't transfered properly")
	}
	_, _, err = p2.TransferTo(p1, 2, &txn)
	if err != nil || p1.GetBalance() != 6 || p2.GetBalance() != 1 {
		t.Error("Pool balance wasn't transfered properly")
	}
	_, _, err = p0.TransferTo(p1, 1, &txn)
	if err != nil || p0.GetBalance() != 8 || p1.GetBalance() != 7 {
		t.Error("Pool balance wasn't transfered properly")
	}
	_, _, err = p0.TransferTo(p2, 8, &txn)
	if err != nil || p0.GetBalance() != 0 || p2.GetBalance() != 9 {
		t.Error("Pool balance wasn't transfered properly")
	}
}
