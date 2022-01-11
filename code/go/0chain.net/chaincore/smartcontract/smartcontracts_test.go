package smartcontract

import (
	"fmt"
	"testing"

	scMocks "0chain.net/chaincore/smartcontract/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/smartcontractinterface/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSmartContracts(t *testing.T) {
	scs := NewSmartContracts()
	require.NotNil(t, scs.v)
}

func TestNewSmartContractsWithVersion(t *testing.T) {
	scsv := NewSmartContractsWithVersion()
	require.NotNil(t, scsv.scs)
}

func TestSmartContractsWithVersion_Get(t *testing.T) {
	type fields struct {
		scs map[string]SmartContractors
	}
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantScs SmartContractors
		wantOk  bool
	}{
		// TODO: Add test cases.
		{
			name: "ok",
			fields: fields{
				scs: map[string]SmartContractors{
					"1.0.0": &scMocks.SmartContractors{},
				},
			},
			args: args{
				version: "1.0.0",
			},
			wantScs: &scMocks.SmartContractors{},
			wantOk:  true,
		},
		{
			name: "version not found",
			fields: fields{
				scs: map[string]SmartContractors{
					"1.0.0": &scMocks.SmartContractors{},
				},
			},
			args: args{
				version: "2.0.0",
			},
			wantScs: nil,
			wantOk:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SmartContractsWithVersion{
				scs: tt.fields.scs,
			}
			gotScs, gotOk := s.Get(tt.args.version)
			assert.Equalf(t, tt.wantScs, gotScs, "Get(%v)", tt.args.version)
			assert.Equalf(t, tt.wantOk, gotOk, "Get(%v)", tt.args.version)
		})
	}
}

type customMockSmartContract struct {
	mocks.SmartContractInterface
	mark int
}

func TestSmartContractsWithVersion_GetSmartContract(t *testing.T) {
	scsV1 := &SmartContracts{
		v: map[string]sci.SmartContractInterface{
			"scAddr1": &customMockSmartContract{mark: 1},
			"scAddr2": &customMockSmartContract{mark: 2},
			"scAddr3": &customMockSmartContract{mark: 3},
		},
	}

	type fields struct {
		scs map[string]SmartContractors
	}
	type args struct {
		version   string
		scAddress string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    sci.SmartContractInterface
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			fields: fields{
				scs: map[string]SmartContractors{
					"1.0.0": scsV1,
				},
			},
			args: args{
				version:   "1.0.0",
				scAddress: "scAddr1",
			},
			want:    scsV1.v["scAddr1"],
			wantErr: assert.NoError,
		},
		{
			name: "version not supported",
			fields: fields{
				scs: map[string]SmartContractors{
					"1.0.0": scsV1,
				},
			},
			args: args{
				version:   "2.0.0",
				scAddress: "scAddr1",
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				return assert.Equal(t, err, ErrSmartContractVersionNotSupported, msg...)
			},
		},
		{
			name: "scAddress not found",
			fields: fields{
				scs: map[string]SmartContractors{
					"1.0.0": scsV1,
				},
			},
			args: args{
				version:   "1.0.0",
				scAddress: "scAddr0",
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				return assert.Equal(t, err, ErrSmartContractNotFound, msg...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SmartContractsWithVersion{
				scs: tt.fields.scs,
			}
			got, err := s.GetSmartContract(tt.args.version, tt.args.scAddress)
			if !tt.wantErr(t, err, fmt.Sprintf("GetSmartContract(%v, %v)", tt.args.version, tt.args.scAddress)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetSmartContract(%v, %v)", tt.args.version, tt.args.scAddress)
		})
	}
}

func TestSmartContractsWithVersion_Register(t *testing.T) {
	scsV1 := &SmartContracts{
		v: map[string]sci.SmartContractInterface{
			"scAddr1": &customMockSmartContract{mark: 1},
			"scAddr2": &customMockSmartContract{mark: 2},
			"scAddr3": &customMockSmartContract{mark: 3},
		},
	}

	type fields struct {
		scs map[string]SmartContractors
	}
	type args struct {
		version string
		scs     SmartContractors
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			fields: fields{
				scs: map[string]SmartContractors{},
			},
			args: args{
				version: "1.0.0",
				scs:     scsV1,
			},
			wantErr: assert.NoError,
		},
		{
			name: "ok 2",
			fields: fields{
				scs: map[string]SmartContractors{
					"1.0.0": scsV1,
				},
			},
			args: args{
				version: "2.0.0",
				scs:     scsV1,
			},
			wantErr: assert.NoError,
		},
		{
			name: "already registered",
			fields: fields{
				scs: map[string]SmartContractors{
					"1.0.0": scsV1,
				},
			},
			args: args{
				version: "1.0.0",
				scs:     scsV1,
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				return assert.Equal(t, err, ErrSmartContractVersionRegistered, msg)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SmartContractsWithVersion{
				scs: tt.fields.scs,
			}
			tt.wantErr(t, s.Register(tt.args.version, tt.args.scs), fmt.Sprintf("Register(%v, %v)", tt.args.version, tt.args.scs))
		})
	}
}

func TestSmartContracts_Get(t *testing.T) {
	scsV1 := &SmartContracts{
		v: map[string]sci.SmartContractInterface{
			"scAddr1": &customMockSmartContract{mark: 1},
			"scAddr2": &customMockSmartContract{mark: 2},
			"scAddr3": &customMockSmartContract{mark: 3},
		},
	}

	type fields struct {
		v map[string]sci.SmartContractInterface
	}
	type args struct {
		scAddress string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantSc sci.SmartContractInterface
		wantOk bool
	}{
		{
			name: "ok",
			fields: fields{
				v: scsV1.v,
			},
			args: args{
				"scAddr1",
			},
			wantSc: scsV1.v["scAddr1"],
			wantOk: true,
		},
		{
			name: "smart contract not found",
			fields: fields{
				v: scsV1.v,
			},
			args: args{
				"scAddr0",
			},
			wantSc: nil,
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scs := &SmartContracts{
				v: tt.fields.v,
			}
			gotSc, gotOk := scs.Get(tt.args.scAddress)
			assert.Equalf(t, tt.wantSc, gotSc, "Get(%v)", tt.args.scAddress)
			assert.Equalf(t, tt.wantOk, gotOk, "Get(%v)", tt.args.scAddress)
		})
	}
}

func TestSmartContracts_GetAll(t *testing.T) {
	scsV1 := &SmartContracts{
		v: map[string]sci.SmartContractInterface{
			"scAddr1": &customMockSmartContract{mark: 1},
			"scAddr2": &customMockSmartContract{mark: 2},
			"scAddr3": &customMockSmartContract{mark: 3},
		},
	}

	type fields struct {
		v map[string]sci.SmartContractInterface
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]sci.SmartContractInterface
	}{
		{
			name: "ok",
			fields: fields{
				v: scsV1.v,
			},
			want: scsV1.v,
		},
		{
			name: "empty",
			fields: fields{
				v: map[string]sci.SmartContractInterface{},
			},
			want: map[string]sci.SmartContractInterface{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scs := &SmartContracts{
				v: tt.fields.v,
			}

			all := scs.GetAll()
			assert.Equalf(t, len(tt.want), len(all), "GetAll() length")

			for k, v := range tt.want {
				assert.Equalf(t, v, all[k], "GetAll() value")
			}
		})
	}
}

func TestSmartContracts_Register(t *testing.T) {
	scsV1 := &SmartContracts{
		v: map[string]sci.SmartContractInterface{
			"scAddr1": &customMockSmartContract{mark: 1},
			"scAddr2": &customMockSmartContract{mark: 2},
			"scAddr3": &customMockSmartContract{mark: 3},
		},
	}

	type fields struct {
		v map[string]sci.SmartContractInterface
	}
	type args struct {
		scAddress string
		sc        sci.SmartContractInterface
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			fields: fields{
				v: map[string]sci.SmartContractInterface{},
			},
			args: args{
				scAddress: "scAddr1",
				sc:        &customMockSmartContract{mark: 1},
			},
			wantErr: assert.NoError,
		},
		{
			name: "already registered",
			fields: fields{
				v: scsV1.v,
			},
			args: args{
				scAddress: "scAddr1",
				sc:        &customMockSmartContract{mark: 1},
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				return assert.ErrorIs(t, err, ErrSmartContractRegistered, msg...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scs := &SmartContracts{
				v: tt.fields.v,
			}
			tt.wantErr(t, scs.Register(tt.args.scAddress, tt.args.sc), fmt.Sprintf("Register(%v, %v)", tt.args.scAddress, tt.args.sc))
		})
	}
}
