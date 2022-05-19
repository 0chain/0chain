package state

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/pkg/currency"

	"github.com/stretchr/testify/assert"

	"0chain.net/core/datastore"
)

func TestNewTransfer(t *testing.T) {
	t.Parallel()

	fromClientID := "from client id"
	toClientID := "to client id"
	amount := currency.Coin(5)

	type args struct {
		fromClientID datastore.Key
		toClientID   datastore.Key
		amount       currency.Coin
	}
	tests := []struct {
		name string
		args args
		want *Transfer
	}{
		{
			name: "OK",
			args: args{
				fromClientID: fromClientID,
				toClientID:   toClientID,
				amount:       amount,
			},
			want: &Transfer{ClientID: fromClientID, ToClientID: toClientID, Amount: amount},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewTransfer(tt.args.fromClientID, tt.args.toClientID, tt.args.amount); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTransfer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransfer_Encode(t *testing.T) {
	t.Parallel()

	tr := NewTransfer("from client id", "to client id", 5)
	blob, err := json.Marshal(tr)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ClientID   datastore.Key
		ToClientID datastore.Key
		Amount     currency.Coin
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields(*tr),
			want:   blob,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &Transfer{
				ClientID:   tt.fields.ClientID,
				ToClientID: tt.fields.ToClientID,
				Amount:     tt.fields.Amount,
			}
			if got := tr.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransfer_Decode(t *testing.T) {
	t.Parallel()

	tr := NewTransfer("from client id", "to client id", 5)
	blob, err := json.Marshal(tr)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ClientID   datastore.Key
		ToClientID datastore.Key
		Amount     currency.Coin
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Transfer
		wantErr bool
	}{
		{
			name:    "OK",
			args:    args{input: blob},
			want:    tr,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &Transfer{
				ClientID:   tt.fields.ClientID,
				ToClientID: tt.fields.ToClientID,
				Amount:     tt.fields.Amount,
			}
			if err := tr.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, tr)
		})
	}
}
