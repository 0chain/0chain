package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"reflect"
	"testing"
)

func TestNotarizationProvider(t *testing.T) {
	tests := []struct {
		name string
		want datastore.Entity
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NotarizationProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NotarizationProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotarization_DoReadLock(t *testing.T) {
	type fields struct {
		NOIDField           datastore.NOIDField
		VerificationTickets []*block.VerificationTicket
		BlockID             datastore.Key
		Round               int64
		Block               *block.Block
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notarization := &Notarization{
				NOIDField:           tt.fields.NOIDField,
				VerificationTickets: tt.fields.VerificationTickets,
				BlockID:             tt.fields.BlockID,
				Round:               tt.fields.Round,
				Block:               tt.fields.Block,
			}
		})
	}
}

func TestNotarization_DoReadUnlock(t *testing.T) {
	type fields struct {
		NOIDField           datastore.NOIDField
		VerificationTickets []*block.VerificationTicket
		BlockID             datastore.Key
		Round               int64
		Block               *block.Block
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notarization := &Notarization{
				NOIDField:           tt.fields.NOIDField,
				VerificationTickets: tt.fields.VerificationTickets,
				BlockID:             tt.fields.BlockID,
				Round:               tt.fields.Round,
				Block:               tt.fields.Block,
			}
		})
	}
}

func TestNotarization_GetEntityMetadata(t *testing.T) {
	type fields struct {
		NOIDField           datastore.NOIDField
		VerificationTickets []*block.VerificationTicket
		BlockID             datastore.Key
		Round               int64
		Block               *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notarization := &Notarization{
				NOIDField:           tt.fields.NOIDField,
				VerificationTickets: tt.fields.VerificationTickets,
				BlockID:             tt.fields.BlockID,
				Round:               tt.fields.Round,
				Block:               tt.fields.Block,
			}
			if got := notarization.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotarization_GetKey(t *testing.T) {
	type fields struct {
		NOIDField           datastore.NOIDField
		VerificationTickets []*block.VerificationTicket
		BlockID             datastore.Key
		Round               int64
		Block               *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notarization := &Notarization{
				NOIDField:           tt.fields.NOIDField,
				VerificationTickets: tt.fields.VerificationTickets,
				BlockID:             tt.fields.BlockID,
				Round:               tt.fields.Round,
				Block:               tt.fields.Block,
			}
			if got := notarization.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupNotarizationEntity(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
