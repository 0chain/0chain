package state

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"testing"

	"0chain.net/chaincore/currency"

	"github.com/stretchr/testify/assert"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

func makeTestState() *State {
	rHash := encryption.RawHash("txn hash")
	return &State{
		TxnHash:      hex.EncodeToString(rHash),
		TxnHashBytes: rHash,
		Round:        1,
		Balance:      5,
		Nonce:        1,
	}
}

func TestState_GetHash(t *testing.T) {
	t.Parallel()

	st := makeTestState()

	type fields struct {
		TxnHash      string
		TxnHashBytes []byte
		Round        int64
		Balance      currency.Coin
		Nonce        int64
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "OK",
			fields: fields(*st),
			want:   util.ToHex(st.GetHashBytes()),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &State{
				TxnHash:      tt.fields.TxnHash,
				TxnHashBytes: tt.fields.TxnHashBytes,
				Round:        tt.fields.Round,
				Balance:      tt.fields.Balance,
				Nonce:        tt.fields.Nonce,
			}
			if got := s.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_GetHashBytes(t *testing.T) {
	t.Parallel()

	st := makeTestState()

	type fields struct {
		TxnHash      string
		TxnHashBytes []byte
		Round        int64
		Balance      currency.Coin
		Nonce        int64
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields(*st),
			want:   encryption.RawHash(st.Encode()),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &State{
				TxnHash:      tt.fields.TxnHash,
				TxnHashBytes: tt.fields.TxnHashBytes,
				Round:        tt.fields.Round,
				Balance:      tt.fields.Balance,
				Nonce:        tt.fields.Nonce,
			}
			if got := s.GetHashBytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHashBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_Encode(t *testing.T) {
	t.Parallel()

	st := makeTestState()

	type fields struct {
		TxnHash      string
		TxnHashBytes []byte
		Round        int64
		Balance      currency.Coin
		Nonce        int64
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields(*st),
			want: func() []byte {
				buf := bytes.NewBuffer(nil)
				buf.Write(st.TxnHashBytes)
				if err := binary.Write(buf, binary.LittleEndian, st.Round); err != nil {
					t.Fatal(err)
				}
				if err := binary.Write(buf, binary.LittleEndian, st.Balance); err != nil {
					t.Fatal(err)
				}

				if err := binary.Write(buf, binary.LittleEndian, st.Nonce); err != nil {
					t.Fatal(err)
				}

				return buf.Bytes()
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &State{
				TxnHash:      tt.fields.TxnHash,
				TxnHashBytes: tt.fields.TxnHashBytes,
				Round:        tt.fields.Round,
				Balance:      tt.fields.Balance,
				Nonce:        tt.fields.Nonce,
			}
			if got := s.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_Decode(t *testing.T) {
	t.Parallel()

	st := makeTestState()
	st.TxnHash = ""
	blob := st.Encode()

	type fields struct {
		TxnHash      string
		TxnHashBytes []byte
		Round        int64
		Balance      currency.Coin
		Nonce        int64
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *State
	}{
		{
			name:    "OK",
			args:    args{data: blob},
			wantErr: false,
			want:    st,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &State{
				TxnHash:      tt.fields.TxnHash,
				TxnHashBytes: tt.fields.TxnHashBytes,
				Round:        tt.fields.Round,
				Balance:      tt.fields.Balance,
				Nonce:        tt.fields.Nonce,
			}

			if err := s.Decode(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestState_ComputeProperties(t *testing.T) {
	t.Parallel()

	st := makeTestState()

	type fields struct {
		TxnHash      string
		TxnHashBytes []byte
		Round        int64
		Balance      currency.Coin
		Nonce        int64
	}
	tests := []struct {
		name   string
		fields fields
		want   *State
	}{
		{
			name: "OK",
			fields: fields{
				TxnHashBytes: st.TxnHashBytes,
				Round:        st.Round,
				Balance:      st.Balance,
				Nonce:        st.Nonce,
			},
			want: st,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &State{
				TxnHash:      tt.fields.TxnHash,
				TxnHashBytes: tt.fields.TxnHashBytes,
				Round:        tt.fields.Round,
				Balance:      tt.fields.Balance,
				Nonce:        tt.fields.Nonce,
			}

			s.ComputeProperties()
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestState_Set(t *testing.T) {
	t.Parallel()

	st := makeTestState()

	type fields struct {
		TxnHash      string
		TxnHashBytes []byte
		Round        int64
		Balance      currency.Coin
		Nonce        int64
	}
	type args struct {
		round   int64
		txnHash string
		nonce   int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *State
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				Balance: 5,
				Nonce:   1,
			},
			args: args{round: st.Round, txnHash: st.TxnHash, nonce: st.Nonce},
			want: st,
		},
		{
			name:    "Invalid_Txn_Hash_ERR",
			args:    args{txnHash: "!"},
			want:    st,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &State{
				TxnHash:      tt.fields.TxnHash,
				TxnHashBytes: tt.fields.TxnHashBytes,
				Round:        tt.fields.Round,
				Balance:      tt.fields.Balance,
				Nonce:        tt.fields.Nonce,
			}

			s.SetRound(tt.args.round)
			err := s.SetTxnHash(tt.args.txnHash)
			if !tt.wantErr {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, s)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
