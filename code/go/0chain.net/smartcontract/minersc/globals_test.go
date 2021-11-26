package minersc_test

import (
	"testing"

	"0chain.net/smartcontract"

	"0chain.net/chaincore/mocks"

	chainstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	. "0chain.net/smartcontract/minersc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGlobalSettings(t *testing.T) {
	require.Len(t, GlobalSettingName, int(NumOfGlobalSettings))
	require.Len(t, GlobalSettingInfo, int(NumOfGlobalSettings))

	for key := range GlobalSettingInfo {
		found := false
		for _, name := range GlobalSettingName {
			if key == name {
				found = true
				break
			}
		}
		require.True(t, found)
	}

	for _, name := range GlobalSettingName {
		_, ok := GlobalSettingInfo[name]
		require.True(t, ok)
	}

}

func TestUpdateGlobals(t *testing.T) {
	const (
		mockNotASetting = "mock not a setting"
	)
	type args struct {
		msc      *MinerSmartContract
		txn      *transaction.Transaction
		input    []byte
		gn       *GlobalNode
		balances chainstate.StateContextI
	}

	type parameters struct {
		client   string
		inputMap map[string]string
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var msc = &MinerSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = &transaction.Transaction{
			ClientID: p.client,
		}
		balances.On("GetTrieNode", GLOBALS_KEY).Return(&GlobalSettings{
			Fields: make(map[string]string),
		}, nil).Once()
		balances.On(
			"InsertTrieNode",
			GLOBALS_KEY,
			mock.MatchedBy(func(globals *GlobalSettings) bool {
				for key, value := range p.inputMap {
					if globals.Fields[key] != value {
						return false
					}
				}
				return true
			}),
		).Return("", nil).Once()

		return args{
			msc:      msc,
			txn:      txn,
			input:    (&smartcontract.StringMap{p.inputMap}).Encode(),
			gn:       &GlobalNode{OwnerId: owner},
			balances: balances,
		}
	}

	type want struct {
		error bool
		msg   string
	}

	testCases := []struct {
		title      string
		parameters parameters
		want       want
	}{
		{
			title: "bad_key",
			parameters: parameters{
				client: owner,
				inputMap: map[string]string{
					mockNotASetting: mockNotASetting,
				},
			},
			want: want{
				error: true,
				msg:   "update_globals: validation: 'mock not a setting' is not a valid global setting",
			},
		},
		{
			title: "all_settings",
			parameters: parameters{
				client: owner,
				inputMap: map[string]string{
					"server_chain.block.min_block_size":                  "1",
					"server_chain.block.max_block_size":                  "10",
					"server_chain.block.max_byte_size":                   "1638400",
					"server_chain.block.replicators":                     "0",
					"server_chain.block.proposal.max_wait_time":          "100ms",
					"server_chain.block.proposal.wait_mode":              "static",
					"server_chain.block.consensus.threshold_by_count":    "66",
					"server_chain.block.consensus.threshold_by_stake":    "0",
					"server_chain.block.sharding.min_active_sharders":    "25",
					"server_chain.block.sharding.min_active_replicators": "25",
					"server_chain.block.validation.batch_size":           "1000",
					"server_chain.block.reuse_txns":                      "false",
					"server_chain.round_range":                           "10000000",
					"server_chain.round_timeouts.softto_min":             "3000",
					"server_chain.round_timeouts.softto_mult":            "3",
					"server_chain.round_timeouts.round_restart_mult":     "2",
					"server_chain.transaction.payload.max_size":          "98304",
					"server_chain.client.signature_scheme":               "bls0chain",
					"server_chain.messages.verification_tickets_to":      "all_miners",
					"server_chain.state.prune_below_count":               "100",
				},
			},
		},
		{
			title: "immutable_key",
			parameters: parameters{
				client: owner,
				inputMap: map[string]string{
					"server_chain.health_check.deep_scan.enabled": "true",
				},
			},
			want: want{
				error: true,
				msg:   "update_globals: validation: server_chain.health_check.deep_scan.enabled is an immutable setting",
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			t.Parallel()
			args := setExpectations(t, test.parameters)

			_, err := args.msc.UpdateGlobals(args.txn, args.input, args.gn, args.balances)
			require.EqualValues(t, test.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.msg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
