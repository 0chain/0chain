package block

import (
	"context"
	"reflect"
	"testing"

	"0chain.net/core/datastore"
)

func TestBlockVerificationTicket_GetEntityMetadata(t *testing.T) {
	type fields struct {
		NOIDField          datastore.NOIDField
		VerificationTicket VerificationTicket
		Round              int64
		BlockID            datastore.Key
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "OK",
			want: bvtEntityMetadata,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bvt := &BlockVerificationTicket{
				NOIDField:          tt.fields.NOIDField,
				VerificationTicket: tt.fields.VerificationTicket,
				Round:              tt.fields.Round,
				BlockID:            tt.fields.BlockID,
			}
			if got := bvt.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockVerificationTicket_GetKey(t *testing.T) {
	type fields struct {
		NOIDField          datastore.NOIDField
		VerificationTicket VerificationTicket
		Round              int64
		BlockID            datastore.Key
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name:   "OK",
			fields: fields{BlockID: "key"},
			want:   "key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bvt := &BlockVerificationTicket{
				NOIDField:          tt.fields.NOIDField,
				VerificationTicket: tt.fields.VerificationTicket,
				Round:              tt.fields.Round,
				BlockID:            tt.fields.BlockID,
			}
			if got := bvt.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockVerificationTicket_Validate(t *testing.T) {
	type fields struct {
		NOIDField          datastore.NOIDField
		VerificationTicket VerificationTicket
		Round              int64
		BlockID            datastore.Key
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			fields:  fields{VerificationTicket: VerificationTicket{VerifierID: "id"}},
			wantErr: false,
		},
		{
			name:    "Empty_ID_ERR",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bvt := &BlockVerificationTicket{
				NOIDField:          tt.fields.NOIDField,
				VerificationTicket: tt.fields.VerificationTicket,
				Round:              tt.fields.Round,
				BlockID:            tt.fields.BlockID,
			}
			if err := bvt.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerificationTicket_GetBlockVerificationTicket(t *testing.T) {
	vt := VerificationTicket{
		VerifierID: "id",
		Signature:  "sign",
	}

	b := NewBlock("", 1)
	b.HashBlock()

	type fields struct {
		VerifierID datastore.Key
		Signature  string
	}
	type args struct {
		b *Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *BlockVerificationTicket
	}{
		{
			name:   "OK",
			fields: fields(vt),
			args:   args{b: b},
			want: &BlockVerificationTicket{
				VerificationTicket: vt,
				BlockID:            b.Hash,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := &VerificationTicket{
				VerifierID: tt.fields.VerifierID,
				Signature:  tt.fields.Signature,
			}
			if got := vt.GetBlockVerificationTicket(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockVerificationTicket() = %v, want %v", got, tt.want)
			}
		})
	}
}
