package tokenpool_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/minersc"
)

const (
	LOCKUPTIME90DAYS = time.Second * 10
	C0               = "client_0"
	C1               = "client_1"
)

type tokenLock struct {
	tokenpool.MockTokenLockInterface
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
}

func (tl tokenLock) IsLocked(entity interface{}) bool {
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		return common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		ts := &tokenStat{Locked: tl.IsLocked(txn)}
		return ts.Encode()
	}
	return nil
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
	p0 := &tokenpool.ZcnLockingPool{}
	p0.TokenLockInterface = &tokenLock{Duration: LOCKUPTIME90DAYS, StartTime: common.Now()}
	if _, _, err := p0.DigPool(C0, txn); err != nil {
		t.Error(err)
	}

	p1 := &tokenpool.ZcnPool{}
	txn.Value = 2
	txn.ClientID = C1
	txn.CreationDate = common.Now()
	if _, _, err := p1.DigPool(C1, txn); err != nil {
		t.Error(err)
	}

	_, _, err := p0.TransferTo(p1, 9, txn)
	if err == nil {
		t.Errorf("transfer happened before lock expired\n\tstart time: %v\n\ttxn time: %v\n", p0.IsLocked(txn), txn.CreationDate)
	}

	time.Sleep(LOCKUPTIME90DAYS + time.Second)
	txn.CreationDate = common.Now()
	_, _, err = p0.TransferTo(p1, 9, txn)
	if err != nil {
		t.Errorf("an error occoured %v\n", err.Error())
	} else if p1.Balance != 11 {
		t.Errorf("pool 1 has wrong balance: %v\ntransaction time: %v\n", p1, common.ToTime(txn.CreationDate))
	}
}

func TestZcnLockingPool_Encode(t *testing.T) {
	t.Parallel()

	zlp := tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "id",
				Balance: 5,
			},
		},
		TokenLockInterface: nil,
	}
	blob, err := json.Marshal(&zlp)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ZcnPool            tokenpool.ZcnPool
		TokenLockInterface tokenpool.TokenLockInterface
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				ZcnPool:            zlp.ZcnPool,
				TokenLockInterface: zlp.TokenLockInterface,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &tokenpool.ZcnLockingPool{
				ZcnPool:            tt.fields.ZcnPool,
				TokenLockInterface: tt.fields.TokenLockInterface,
			}
			if got := p.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZcnLockingPool_Decode(t *testing.T) {
	t.Parallel()

	zlp := tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "id",
				Balance: 5,
			},
		},
		TokenLockInterface: nil,
	}
	blob, err := json.Marshal(&zlp)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ZcnPool            tokenpool.ZcnPool
		TokenLockInterface tokenpool.TokenLockInterface
	}
	type args struct {
		input    []byte
		tokelock tokenpool.TokenLockInterface
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *tokenpool.ZcnLockingPool
	}{
		{
			name:    "OK",
			args:    args{input: blob, tokelock: zlp.TokenLockInterface},
			want:    &zlp,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &tokenpool.ZcnLockingPool{
				ZcnPool:            tt.fields.ZcnPool,
				TokenLockInterface: tt.fields.TokenLockInterface,
			}
			if err := p.Decode(tt.args.input, tt.args.tokelock); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestZcnLockingPool_GetBalance(t *testing.T) {
	t.Parallel()

	type fields struct {
		ZcnPool            tokenpool.ZcnPool
		TokenLockInterface tokenpool.TokenLockInterface
	}
	tests := []struct {
		name   string
		fields fields
		want   currency.Coin
	}{
		{
			name: "OK",
			want: 5,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &tokenpool.ZcnLockingPool{
				ZcnPool:            tt.fields.ZcnPool,
				TokenLockInterface: tt.fields.TokenLockInterface,
			}

			p.SetBalance(tt.want)
			if got := p.GetBalance(); got != tt.want {
				t.Errorf("GetBalance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZcnLockingPool_GetID(t *testing.T) {
	t.Parallel()

	id := "id"

	type fields struct {
		ZcnPool            tokenpool.ZcnPool
		TokenLockInterface tokenpool.TokenLockInterface
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				ZcnPool: tokenpool.ZcnPool{
					tokenpool.TokenPool{
						ID: id,
					},
				},
			},
			want: id,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &tokenpool.ZcnLockingPool{
				ZcnPool:            tt.fields.ZcnPool,
				TokenLockInterface: tt.fields.TokenLockInterface,
			}
			if got := p.GetID(); got != tt.want {
				t.Errorf("GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZcnLockingPool_FillPool(t *testing.T) {
	t.Parallel()

	txn := transaction.Transaction{}
	txn.Value = 5
	txn.ClientID = "client id"
	txn.ClientID = "to client id"

	p := tokenpool.ZcnPool{}
	p.Balance += currency.Coin(txn.Value)
	p.ID = "pool id"

	tpr := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		FromClient: txn.ClientID,
		ToPool:     p.ID,
		ToClient:   txn.ToClientID,
		Value:      currency.Coin(txn.Value),
	}
	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, currency.Coin(txn.Value))

	type fields struct {
		ZcnPool            tokenpool.ZcnPool
		TokenLockInterface tokenpool.TokenLockInterface
	}
	type args struct {
		txn *transaction.Transaction
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *state.Transfer
		want1   string
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				ZcnPool: p,
			},
			args:    args{txn: &txn},
			want:    transfer,
			want1:   string(tpr.Encode()),
			wantErr: false,
		},
		{
			name: "ERR",
			fields: fields{
				ZcnPool: p,
			},
			args:    args{txn: &transaction.Transaction{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &tokenpool.ZcnLockingPool{
				ZcnPool:            tt.fields.ZcnPool,
				TokenLockInterface: tt.fields.TokenLockInterface,
			}
			got, got1, err := p.FillPool(tt.args.txn)
			if (err != nil) != tt.wantErr {
				t.Errorf("FillPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FillPool() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("FillPool() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestZcnLockingPool_DrainPool(t *testing.T) {
	t.Parallel()

	var (
		zlp = tokenpool.ZcnLockingPool{
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      "id",
					Balance: 5,
				},
			},
			TokenLockInterface: &minersc.ViewChangeLock{},
		}
		fromClientID = "from client id"
		toClientID   = "to client id"
		value        = currency.Coin(4)

		tpr = &tokenpool.TokenPoolTransferResponse{
			FromClient: fromClientID,
			ToClient:   toClientID,
			Value:      value,
			FromPool:   zlp.TokenPool.ID,
		}
		transfer = state.NewTransfer(fromClientID, toClientID, value)
	)

	type fields struct {
		ZcnPool            tokenpool.ZcnPool
		TokenLockInterface tokenpool.TokenLockInterface
	}
	type args struct {
		fromClientID string
		toClientID   string
		value        currency.Coin
		entity       interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *state.Transfer
		want1   string
		wantErr bool
	}{
		{
			name: "Locked_Entity_ERR",
			fields: fields{
				ZcnPool:            zlp.ZcnPool,
				TokenLockInterface: zlp.TokenLockInterface,
			},
			args: args{
				entity: int64(5),
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				ZcnPool:            zlp.ZcnPool,
				TokenLockInterface: zlp.TokenLockInterface,
			},
			args: args{
				value:        value,
				toClientID:   toClientID,
				fromClientID: fromClientID,
			},
			want:    transfer,
			want1:   string(tpr.Encode()),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &tokenpool.ZcnLockingPool{
				ZcnPool:            tt.fields.ZcnPool,
				TokenLockInterface: tt.fields.TokenLockInterface,
			}
			got, got1, err := p.DrainPool(tt.args.fromClientID, tt.args.toClientID, tt.args.value, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("DrainPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DrainPool() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("DrainPool() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestZcnLockingPool_EmptyPool(t *testing.T) {
	t.Parallel()

	var (
		zlp = tokenpool.ZcnLockingPool{
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      "id",
					Balance: 5,
				},
			},
			TokenLockInterface: &minersc.ViewChangeLock{},
		}
		fromClientID = "from client id"
		toClientID   = "to client id"

		tpr = &tokenpool.TokenPoolTransferResponse{
			FromClient: fromClientID,
			ToClient:   toClientID,
			Value:      zlp.Balance,
			FromPool:   zlp.TokenPool.ID,
		}
		transfer = state.NewTransfer(fromClientID, toClientID, zlp.Balance)
	)

	type fields struct {
		ZcnPool            tokenpool.ZcnPool
		TokenLockInterface tokenpool.TokenLockInterface
	}
	type args struct {
		fromClientID string
		toClientID   string
		entity       interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *state.Transfer
		want1   string
		wantErr bool
	}{
		{
			name: "Locked_Entity_ERR",
			fields: fields{
				ZcnPool:            zlp.ZcnPool,
				TokenLockInterface: zlp.TokenLockInterface,
			},
			args: args{
				entity: int64(5),
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				ZcnPool:            zlp.ZcnPool,
				TokenLockInterface: zlp.TokenLockInterface,
			},
			args: args{
				toClientID:   toClientID,
				fromClientID: fromClientID,
			},
			want:    transfer,
			want1:   string(tpr.Encode()),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &tokenpool.ZcnLockingPool{
				ZcnPool:            tt.fields.ZcnPool,
				TokenLockInterface: tt.fields.TokenLockInterface,
			}
			got, got1, err := p.EmptyPool(tt.args.fromClientID, tt.args.toClientID, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmptyPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EmptyPool() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("EmptyPool() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
