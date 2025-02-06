package minersc_test

import (
	"testing"

	"0chain.net/core/config"
	"0chain.net/smartcontract/minersc"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateGlobals(t *testing.T) {
	const (
		mockNotASetting = "mock not a setting"
		mockRoundNumber = 17
	)
	type args struct {
		msc      *minersc.MinerSmartContract
		txn      *transaction.Transaction
		input    []byte
		gn       *minersc.GlobalNode
		balances chainstate.StateContextI
	}

	type parameters struct {
		client   string
		inputMap map[string]string
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var msc = &minersc.MinerSmartContract{
			SmartContract: sci.NewSC(minersc.ADDRESS),
		}
		var txn = &transaction.Transaction{
			ClientID: p.client,
		}
		balances.On("GetTrieNode", minersc.GLOBALS_KEY, mock.AnythingOfType("*minersc.GlobalSettings")).Return(nil).Once()
		balances.On(
			"InsertTrieNode",
			minersc.GLOBALS_KEY,
			mock.MatchedBy(func(globals *minersc.GlobalSettings) bool {
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
			input:    (&config.StringMap{p.inputMap}).Encode(),
			gn:       minersc.NewGlobalNode(owner, make(map[string]int)),
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
				msg:   "update_globals: validation: server_chain.health_check.deep_scan.enabled cannot be modified via a transaction",
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
