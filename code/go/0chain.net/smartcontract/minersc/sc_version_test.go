package minersc

import (
	"errors"
	"fmt"
	"testing"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStateContextI struct {
	mocks.StateContextI
	scVersion string
	err       error
}

func (s *mockStateContextI) InsertTrieNode(key datastore.Key, node util.Serializable) (datastore.Key, error) {
	if s.err != nil {
		return "", s.err
	}

	vn := node.(*SCVersionNode)
	s.scVersion = vn.String()
	return "", nil
}

func TestMinerSmartContract_updateSCVersion(t *testing.T) {
	updateSCVersionReqV1 := UpdateSCVersionTxn{Version: "1.0.0"}
	txnDataV1, err := updateSCVersionReqV1.Encode()
	require.NoError(t, err)

	updateSCVersionReqV2 := UpdateSCVersionTxn{Version: "2.0.0"}
	txnDataV2, err := updateSCVersionReqV2.Encode()
	require.NoError(t, err)

	prepareStateContext := func() state.StateContextI {
		stateCtx := &mockStateContextI{}
		scV, err := semver.Make("1.0.0")
		require.NoError(t, err)
		stateCtx.On("GetSCVersion").Return(scV)

		scV2, err := semver.Make("2.0.0")
		require.NoError(t, err)
		stateCtx.On("CanUpdateSCVersion").Return(&scV2, true)
		return stateCtx
	}

	prepareNotAllowedStateContext := func() state.StateContextI {
		stateCtx := &mockStateContextI{}
		scV, err := semver.Make("1.0.0")
		require.NoError(t, err)
		stateCtx.On("GetSCVersion").Return(scV)

		stateCtx.On("CanUpdateSCVersion").Return(nil, false)
		return stateCtx
	}

	type args struct {
		t         *transaction.Transaction
		inputData []byte
		balances  func() state.StateContextI
	}
	tests := []struct {
		name     string
		args     args
		wantResp string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			args: args{
				t: &transaction.Transaction{
					ClientID: owner,
				},
				inputData: txnDataV2,
				balances:  prepareStateContext,
			},
			wantResp: "2.0.0",
			wantErr:  assert.NoError,
		},
		{
			name: "not allowed update yet",
			args: args{
				t: &transaction.Transaction{
					ClientID: owner,
				},
				inputData: txnDataV1,
				balances:  prepareNotAllowedStateContext,
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				assert.ErrorIs(t, common.NewError("update_sc_version_not_allowed", ""), err, msg...)
				return false
			},
		},
		{
			name: "not allowed, incorrect version",
			args: args{
				t: &transaction.Transaction{
					ClientID: owner,
				},
				inputData: txnDataV1,
				balances:  prepareStateContext,
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				assert.ErrorIs(t, common.NewError("update_sc_version_not_allowed", ""), err, msg...)
				return false
			},
		},
		{
			name: "unauthorized access",
			args: args{
				t: &transaction.Transaction{
					ClientID: "not_owner_id",
				},
				inputData: txnDataV2,
				balances:  prepareStateContext,
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				assert.ErrorIs(t, common.NewError("update_sc_version_unauthorized_access", ""), err, msg...)
				return false
			},
		},
		{
			name: "invalid txn data",
			args: args{
				t: &transaction.Transaction{
					ClientID: owner,
				},
				inputData: []byte("invalid txn data"),
				balances:  prepareStateContext,
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				assert.ErrorIs(t, common.NewError("update_sc_version_invalid_txn_input", ""), err, msg...)
				return false
			},
		},
		{
			name: "invalid version data",
			args: args{
				t: &transaction.Transaction{
					ClientID: owner,
				},
				inputData: []byte(`{"version": "abc"}`),
				balances:  prepareStateContext,
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				assert.ErrorIs(t, common.NewError("update_sc_version_invalid_version", ""), err, msg...)
				return false
			},
		},
		{
			name: "save node failed",
			args: args{
				t: &transaction.Transaction{
					ClientID: owner,
				},
				inputData: []byte(`{"version": "2.0.0"}`),
				balances: func() state.StateContextI {
					stateCtx := &mockStateContextI{err: errors.New("save failed")}
					scV, err := semver.Make("1.0.0")
					require.NoError(t, err)
					stateCtx.On("GetSCVersion").Return(scV)

					scV2, err := semver.Make("2.0.0")
					require.NoError(t, err)
					stateCtx.On("CanUpdateSCVersion").Return(&scV2, true)

					return stateCtx
				},
			},
			wantErr: func(t assert.TestingT, err error, msg ...interface{}) bool {
				assert.ErrorIs(t, common.NewError("update_sc_version_save_error", ""), err, msg...)
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msc := &MinerSmartContract{}
			sctx := tt.args.balances()
			gotResp, err := msc.updateSCVersion(tt.args.t, tt.args.inputData, nil, sctx)
			if !tt.wantErr(t, err, fmt.Sprintf("updateSCVersion(%v, %v, %v, %v)", tt.args.t, tt.args.inputData, nil, sctx)) {
				return
			}
			assert.Equalf(t, tt.wantResp, gotResp, "updateSCVersion(%v, %v, %v, %v)", tt.args.t, tt.args.inputData, nil, sctx)
			ss, ok := sctx.(*mockStateContextI)
			require.True(t, ok)
			require.Equal(t, tt.wantResp, ss.scVersion)
		})
	}
}
