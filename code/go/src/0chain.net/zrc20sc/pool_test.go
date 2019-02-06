package zrc20sc

import (
	"testing"

	"0chain.net/transaction"
)

const (
	clientID0      = "client0_address"
	clientID1      = "client1_address"
	zrc20scAddress = "zrc20sc_address"
)

func TestDigPool(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 4}
	transfer, _, err := pool0.DigPool(clientID0, txn)
	if pool0.Balance != 2 || err != nil {
		t.Errorf("pool balance should be 2, instead it is %v\n", pool0.Balance)
	}
	if transfer.Amount != 3 {
		t.Errorf("wrong amount taken from client, should be 3, was %v\n", transfer.Amount)
	}
	pool0 = &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn = &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 99}
	transfer, _, err = pool0.DigPool(clientID0, txn)
	if pool0.Balance != 66 || err != nil {
		t.Errorf("pool balance should be 2, instead it is %v\n", pool0.Balance)
	}
	if transfer.Amount != 99 {
		t.Errorf("wrong amount taken from client, should be 3, was %v\n", transfer.Amount)
	}
}

func TestDigPoolInsufficentFundsForExchange(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 2}
	transfer, _, err := pool0.DigPool(clientID0, txn)
	if pool0.Balance != 0 {
		t.Errorf("pool balance should be 2, instead it is %v\n", pool0.Balance)
	}
	if transfer != nil {
		t.Error("The transfer should be nil")
	}
	if err == nil {
		t.Error("The error shouldn't be nil")
	}
	t.Logf("error: %v\n", err.Error())
}

func TestFillPool(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 4}
	transfer, _, _ := pool0.DigPool(clientID0, txn)
	if transfer.Amount != 3 || pool0.Balance != 2 {
		t.Errorf("transfer should be 3, but is %v\npool balance should be 2, but is %v\n", transfer.Amount, pool0.Balance)
	}
	txn = &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 13}
	transfer, _, _ = pool0.FillPool(txn)
	if transfer.Amount != 12 || pool0.Balance != 10 {
		t.Errorf("transfer should be 3, but is %v\npool balance should be 2, but is %v\n", transfer.Amount, pool0.Balance)
	}
}

func TestFillPoolInsufficentFunds(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 4}
	transfer, _, _ := pool0.DigPool(clientID0, txn)
	if transfer.Amount != 3 || pool0.Balance != 2 {
		t.Errorf("transfer should be 3, but is %v\npool balance should be 2, but is %v\n", transfer.Amount, pool0.Balance)
	}
	txn = &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 2}
	_, _, err := pool0.FillPool(txn)
	if err == nil {
		t.Error("The error shouldn't be nil")
	}
	t.Logf("error: %v\n", err.Error())
}

func TestTransferTo(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	pool1 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 4}
	txn1 := &transaction.Transaction{ClientID: clientID1, ToClientID: zrc20scAddress, Value: 4}
	pool0.DigPool(clientID0, txn0)
	pool1.DigPool(clientID1, txn1)
	pool0.TransferTo(pool1, 2, txn0)
	if pool0.Balance != 0 || pool1.Balance != 4 {
		t.Errorf("pool0 balance should be 0, but is %v\npool1 balance should be 4, but is %v\n", pool0.Balance, pool1.Balance)
	}
}

func TestTransferToInsufficentFunds(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	pool1 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 4}
	txn1 := &transaction.Transaction{ClientID: clientID1, ToClientID: zrc20scAddress, Value: 4}
	pool0.DigPool(clientID0, txn0)
	pool1.DigPool(clientID1, txn1)
	pool0.TransferTo(pool1, 3, txn0)
	if pool0.Balance != 2 || pool1.Balance != 2 {
		t.Errorf("pool0 balance should be 0, but is %v\npool1 balance should be 4, but is %v\n", pool0.Balance, pool1.Balance)
	}
}

func TestInterPoolTransfer(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_0", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	pool1 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_1", ExchangeRate: tokenRatio{ZCN: 7, Other: 5}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 9}
	txn1 := &transaction.Transaction{ClientID: clientID1, ToClientID: zrc20scAddress, Value: 7}
	pool0.DigPool(clientID0, txn0)
	pool1.DigPool(clientID1, txn1)
	transfer, _, _ := pool0.TransferTo(pool1, 6, txn0)
	if pool0.Balance != 0 || pool1.Balance != 10 || transfer.Amount != 2 {
		t.Errorf("pool0 balance should be 0, but is %v\npool1 balance should be 4, but is %v\n", pool0.Balance, pool1.Balance)
	}
}

func TestInterPoolTransferInsufficentFundsFromPool(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_0", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	pool1 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_1", ExchangeRate: tokenRatio{ZCN: 7, Other: 5}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 9}
	txn1 := &transaction.Transaction{ClientID: clientID1, ToClientID: zrc20scAddress, Value: 7}
	pool0.DigPool(clientID0, txn0)
	pool1.DigPool(clientID1, txn1)
	_, _, err := pool0.TransferTo(pool1, 1, txn0)
	if err == nil {
		t.Error("The error shouldn't be nil")
	}
	t.Logf("error: %v\n", err.Error())
}

func TestInterPoolTransferInsufficentFundsToPool(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_0", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	pool1 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_1", ExchangeRate: tokenRatio{ZCN: 7, Other: 5}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 9}
	txn1 := &transaction.Transaction{ClientID: clientID1, ToClientID: zrc20scAddress, Value: 7}
	pool0.DigPool(clientID0, txn0)
	pool1.DigPool(clientID1, txn1)
	_, _, err := pool0.TransferTo(pool1, 2, txn0)
	if err == nil {
		t.Error("The error shouldn't be nil")
	}
	t.Logf("error: %v\n", err.Error())
}

func TestDrainPool(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_0", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 9}
	pool0.DigPool(clientID0, txn0)
	transfer, _, _ := pool0.DrainPool(zrc20scAddress, clientID0, 3)
	if pool0.Balance != 4 || transfer.Amount != 3 {
		t.Errorf("pool0 balance should be 4, but is %v\ntransfer amount should be 3, but is %v\n", pool0.Balance, transfer.Amount)
	}
}

func TestDrainPoolInsufficentFunds(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_0", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 9}
	pool0.DigPool(clientID0, txn0)
	_, _, err := pool0.DrainPool(zrc20scAddress, clientID0, 7)
	if err == nil {
		t.Error("The error shouldn't be nil")
	}
	t.Logf("error: %v\n", err.Error())
}

func TestEmptyPool(t *testing.T) {
	pool0 := &zrc20Pool{tokenInfo: tokenInfo{TokenName: "test_token_0", ExchangeRate: tokenRatio{ZCN: 3, Other: 2}}}
	txn0 := &transaction.Transaction{ClientID: clientID0, ToClientID: zrc20scAddress, Value: 9}
	pool0.DigPool(clientID0, txn0)
	transfer, _, _ := pool0.EmptyPool(zrc20scAddress, clientID0)
	if pool0.Balance != 0 || transfer.Amount != 9 {
		t.Errorf("pool0 balance should be 0, but is %v\ntransfer amount should be 9, but is %v\n", pool0.Balance, transfer.Amount)
	}
}
