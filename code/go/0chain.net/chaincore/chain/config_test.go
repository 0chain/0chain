package chain

import (
	"bytes"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/minersc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"testing"
)

func TestUpdate(t *testing.T) {
	type args struct {
		config  config.ChainConfig
		updates minersc.GlobalSettings
	}

	type parameters struct {
		updates    minersc.GlobalSettings
		configType string
		zChainYaml []byte
	}
	type want struct {
		result config.ChainConfig
	}
	setExpectations := func(t *testing.T, p parameters, w *want) args {
		viper.SetConfigType(p.configType)
		err := viper.ReadConfig(bytes.NewBuffer(p.zChainYaml))
		require.NoError(t, err)
		chain := NewChainFromConfig()

		return args{
			config:  chain.ChainConfig,
			updates: p.updates,
		}
	}

	testCases := []struct {
		title      string
		parameters parameters
		want       want
	}{
		{
			title: "ok_no_chainge",
			parameters: parameters{
				updates: minersc.GlobalSettings{
					Fields: map[string]string{
						//"server_chain.owner":                                 mock_owner,
						"server_chain.block.max_block_size":                  "10",
						"server_chain.block.min_block_size":                  "1",
						"server_chain.block.max_byte_size":                   "1638400",
						"server_chain.block.min_generators":                  "2",
						"server_chain.block.generators_percent":              "0.2",
						"server_chain.block.replicators":                     "0",
						"server_chain.block.consensus.threshold_by_count":    "66",
						"server_chain.block.consensus.threshold_by_stake":    "0",
						"server_chain.block.validation.batch_size":           "1000",
						"server_chain.transaction.payload.max_size":          "98304",
						"server_chain.state.prune_below_count":               "100",
						"server_chain.round_range":                           "10000000",
						"server_chain.messages.verification_tickets_to":      "all_miners",
						"server_chain.health_check.show_counters":            "true",
						"server_chain.block.proposal.max_wait_time":          "180ms",
						"server_chain.block.proposal.wait_mode":              "static",
						"server_chain.block.reuse_txns":                      "false",
						"server_chain.block.sharding.min_active_sharders":    "25",
						"server_chain.block.sharding.min_active_replicators": "25",
						"server_chain.smart_contract.timeout":                "8000ms",
						"server_chain.round_timeouts.softto_min":             "1500",
						"server_chain.round_timeouts.softto_mult":            "1",
						"server_chain.round_timeouts.round_restart_mult":     "10",
						"server_chain.client.signature_scheme":               "bls0chain",
					},
				},
				configType: "yaml",
				zChainYaml: []byte(exampleZChainYaml),
			},
		},
		{
			title: "ok_unknown_entry",
			parameters: parameters{
				updates: minersc.GlobalSettings{
					Fields: map[string]string{
						"server_chain.block.generation.timeout": "17",
					},
				},
				configType: "yaml",
				zChainYaml: []byte(exampleZChainYaml),
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			args := setExpectations(t, test.parameters, &test.want)
			before := args.config

			err := args.config.Update(args.updates.Fields, args.updates.Version)
			require.NoError(t, err)
			require.EqualValues(t, before, args.config)
		})
	}
}

const exampleZChainYaml string = `
development:
  state: true
  dkg: true
  view_change: false
  block_rewards: false
  smart_contract:
    storage: true
    faucet: true
    interest: true
    miner: true
    multisig: true
    vesting: true
  txn_generation:
    wallets: 50
    max_transactions: 0
    max_txn_fee: 10000
    min_txn_fee: 0
    max_txn_value: 10000000000
    min_txn_value: 100
  faucet:
    refill_amount: 1000000000000000
server_chain:
  id: "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe"
  owner: "edb90b850f2e7e7cbd0a1fa370fdcc5cd378ffbec95363a7bc0e5a98b8ba5759"
  decimals: 10
  tokens: 200000000
  genesis_block:
    id: "ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae4"
  block:
    min_block_size: 1
    max_block_size: 10
    max_byte_size: 1638400
    min_generators: 2
    generators_percent: 0.2
    replicators: 0
    generation:
      timeout: 15
      retry_wait_time: 5 #milliseconds
    proposal:
      max_wait_time: 180ms # milliseconds
      wait_mode: static # static or dynamic
    consensus:
      threshold_by_count: 66 # percentage (registration)
      threshold_by_stake: 0 # percent
    sharding:
      min_active_sharders: 25 # percentage
      min_active_replicators: 25 # percentage
    validation:
      batch_size: 1000
    reuse_txns: false
    storage:
      provider: blockstore.FSBlockStore # blockstore.FSBlockStore or blockstore.BlockDBStore
  round_range: 10000000
  round_timeouts:
    softto_min: 1500 # in miliseconds
    softto_mult: 1 # multiples of mean network time (mnt)  softto = max{softo_min, softto_mult * mnt}
    round_restart_mult: 10 # number of soft timeouts before round is restarted
    timeout_cap: 0 # 0 indicates no cap
    vrfs_timeout_mismatch_tolerance: 5
  transaction:
    payload:
      max_size: 98304 # bytes
    timeout: 30s # seconds
    min_fee: 0
  client:
    signature_scheme: bls0chain # ed25519 or bls0chain
    discover: true
  messages:
    verification_tickets_to: all_miners # generator or all_miners
  state:
    prune_below_count: 100 # rounds
    sync:
      timeout: 10s # seconds
  stuck:
    check_interval: 10s # seconds
    time_threshold: 60s #seconds
  smart_contract:
    timeout: 8000ms # milliseconds
  health_check:
    show_counters: true
    deep_scan:
      enabled: false
      settle_secs: 30
      window: 0 #Full scan till round 0
      repeat_interval_mins: 3m #minutes
      report_status_mins: 1m #minutes
      batch_size: 50
    proximity_scan:
      enabled: true
      settle_secs: 30
      window: 100000 #number of blocks, Do not make 0 with minio ON, Should be less than minio old block round range
      repeat_interval_mins: 1m #minutes
      report_status_mins: 1m #minutes
      batch_size: 50
  lfb_ticket:
    rebroadcast_timeout: "15s" #
    ahead: 5 # should be >= 5
    fb_fetching_lifetime: 10s #
  async_blocks_fetching:
    max_simultaneous_from_miners: 100
    max_simultaneous_from_sharders: 30
`
