package stakepool

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/logging"

	"0chain.net/chaincore/transaction"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
)

func init() {
	logging.InitLogging("development", "")
}

func TestStakePool_DistributeRewards(t *testing.T) {
	providerID := "provider_id"
	providerType := spenum.Blobber
	type args struct {
		value              currency.Coin
		numDelegates       int
		delegateBal        []currency.Coin
		serviceChargeRatio float64
	}

	type want struct {
		poolReward      currency.Coin
		delegateRewards []currency.Coin
		eventTags       []event.EventTag
		err             bool
		errMsg          string
	}

	setup := func(t *testing.T, arg args, w want) (*StakePool, state.StateContextI, map[string]currency.Coin) {
		var (
			balances        = newTestBalances(t, false)
			sp              = NewStakePool()
			DelegateRewards = make(map[string]currency.Coin)
		)
		require.GreaterOrEqual(t, arg.serviceChargeRatio, float64(0))
		require.LessOrEqual(t, arg.serviceChargeRatio, float64(1))

		for i := 0; i < arg.numDelegates; i++ {
			delegateId := "delegate_" + strconv.Itoa(i)
			sp.Pools[delegateId] = &DelegatePool{
				DelegateID: delegateId,
				Balance:    arg.delegateBal[i],
			}
			if arg.serviceChargeRatio < 1 && len(w.eventTags) > 0 {
				DelegateRewards[delegateId] = w.delegateRewards[i]
			}
			sp.Settings.ServiceChargeRatio = arg.serviceChargeRatio
		}

		return sp, balances, DelegateRewards
	}

	validate := func(t *testing.T, sp *StakePool, want want) {
		require.EqualValues(t, want.poolReward, sp.Reward)
		for i := range want.delegateRewards {
			delegateId := "delegate_" + strconv.Itoa(i)
			delegateWallet, ok := sp.Pools[delegateId]
			require.True(t, ok)
			require.EqualValues(t, want.delegateRewards[i], delegateWallet.Reward)
		}
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "0 value",
			args: args{
				value:              0,
				numDelegates:       2,
				delegateBal:        []currency.Coin{10, 12},
				serviceChargeRatio: 0.3,
			},
			want: want{
				poolReward:      0,
				delegateRewards: []currency.Coin{0, 0},
				eventTags:       []event.EventTag{},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "value less that delegate numbers, 0 service charge ratio",
			args: args{
				value:              1,
				numDelegates:       4,
				delegateBal:        []currency.Coin{15, 11, 18, 21},
				serviceChargeRatio: 0,
			},
			want: want{
				poolReward:      0,
				delegateRewards: []currency.Coin{1, 0, 0, 0},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "value less that delegate numbers with service charge ratio",
			args: args{
				value:              4,
				numDelegates:       5,
				delegateBal:        []currency.Coin{15, 11, 18, 21, 10},
				serviceChargeRatio: 0.3,
			},
			want: want{
				poolReward:      1,
				delegateRewards: []currency.Coin{1, 1, 1, 0, 0},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "value getting equally distributed, 0 service charge",
			args: args{
				value:              100,
				numDelegates:       5,
				delegateBal:        []currency.Coin{1, 1, 1, 1, 1},
				serviceChargeRatio: 0,
			},
			want: want{
				poolReward:      0,
				delegateRewards: []currency.Coin{20, 20, 20, 20, 20},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "no delegate stake",
			args: args{
				value:              1,
				numDelegates:       4,
				delegateBal:        []currency.Coin{0, 0, 0, 0},
				serviceChargeRatio: 0.1,
			},
			want: want{
				poolReward:      0,
				delegateRewards: []currency.Coin{0, 0, 0, 0},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             true,
				errMsg:          "no stake",
			},
		},
		{
			name: "0 value is lost with unequal delegate distribution",
			args: args{
				value:              50,
				numDelegates:       2,
				delegateBal:        []currency.Coin{13, 19},
				serviceChargeRatio: 0.5,
			},
			want: want{
				poolReward:      25,
				delegateRewards: []currency.Coin{11, 14},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "100 percent service charge",
			args: args{
				value:              50,
				numDelegates:       2,
				delegateBal:        []currency.Coin{13, 19},
				serviceChargeRatio: 1,
			},
			want: want{
				poolReward:      50,
				delegateRewards: []currency.Coin{0, 0},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "no delegates with 0 service charge",
			args: args{
				value:              100,
				numDelegates:       0,
				delegateBal:        []currency.Coin{},
				serviceChargeRatio: 0,
			},
			want: want{
				poolReward:      100,
				delegateRewards: []currency.Coin{},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "single delegates with 1 coin and 50 percent service charge",
			args: args{
				value:              1,
				numDelegates:       1,
				delegateBal:        []currency.Coin{1},
				serviceChargeRatio: 0.5,
			},
			want: want{
				poolReward:      0,
				delegateRewards: []currency.Coin{1},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "single delegates with 100 percent service charge",
			args: args{
				value:              20,
				numDelegates:       1,
				delegateBal:        []currency.Coin{100},
				serviceChargeRatio: 1,
			},
			want: want{
				poolReward:      20,
				delegateRewards: []currency.Coin{0},
				eventTags:       []event.EventTag{event.TagStakePoolReward},
				err:             false,
				errMsg:          "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp, balances, DelegateRewards := setup(t, tt.args, tt.want)
			err := sp.DistributeRewards(tt.args.value, providerID, providerType, spenum.BlockRewardBlobber, balances)
			require.EqualValues(t, tt.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errMsg, err.Error())
			}
			events := balances.GetEvents()
			require.EqualValues(t, len(tt.want.eventTags), len(events))
			if len(tt.want.eventTags) > 0 {
				for i, event := range events {
					require.EqualValues(t, tt.want.eventTags[i], event.Tag)
					require.EqualValues(t, event.Data, &dbs.StakePoolReward{
						ProviderID: dbs.ProviderID{
							ID:   providerID,
							Type: providerType,
						},
						Reward:            tt.want.poolReward,
						DelegateRewards:   DelegateRewards,
						DelegatePenalties: make(map[string]currency.Coin),
						RewardType:        spenum.BlockRewardBlobber,
						AllocationID:      "",
						DelegateWallet:    "",
					})
				}
			}
			validate(t, sp, tt.want)
		})
	}
}

func TestStakePool_DistributeRewardsRandN(t *testing.T) {
	providerID := "provider_id"
	providerType := spenum.Blobber
	var RoundRandomSeed int64 = 839695260482366273
	var NumMinerDelegatesRewarded int = 10
	type args struct {
		value              currency.Coin
		numDelegates       int
		delegateBal        []currency.Coin
		serviceChargeRatio float64
	}

	type want struct {
		poolReward currency.Coin
		eventTags  []event.EventTag
		err        bool
		errMsg     string
	}

	setup := func(t *testing.T, arg args) (*StakePool, state.StateContextI) {
		var (
			balances = newTestBalances(t, false)
			sp       = NewStakePool()
		)
		require.GreaterOrEqual(t, arg.serviceChargeRatio, float64(0))
		require.LessOrEqual(t, arg.serviceChargeRatio, float64(1))

		for i := 0; i < arg.numDelegates; i++ {
			delegateId := "delegate_" + strconv.Itoa(i)
			sp.Pools[delegateId] = &DelegatePool{
				DelegateID: delegateId,
				Balance:    arg.delegateBal[i],
			}
			sp.Settings.ServiceChargeRatio = arg.serviceChargeRatio
		}

		return sp, balances
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "0 value",
			args: args{
				value:              0,
				numDelegates:       2,
				delegateBal:        []currency.Coin{10, 12},
				serviceChargeRatio: 0.3,
			},
			want: want{
				poolReward: 0,
				eventTags:  []event.EventTag{},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "value less that delegate numbers, 0 service charge ratio",
			args: args{
				value:              1,
				numDelegates:       4,
				delegateBal:        []currency.Coin{15, 11, 18, 21},
				serviceChargeRatio: 0,
			},
			want: want{
				poolReward: 0,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "value less that delegate numbers with service charge ratio",
			args: args{
				value:              4,
				numDelegates:       5,
				delegateBal:        []currency.Coin{15, 11, 18, 21, 10},
				serviceChargeRatio: 0.3,
			},
			want: want{
				poolReward: 1,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "value getting equally distributed, 0 service charge",
			args: args{
				value:              100,
				numDelegates:       5,
				delegateBal:        []currency.Coin{1, 1, 1, 1, 1},
				serviceChargeRatio: 0,
			},
			want: want{
				poolReward: 0,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "no delegate stake",
			args: args{
				value:              1,
				numDelegates:       4,
				delegateBal:        []currency.Coin{0, 0, 0, 0},
				serviceChargeRatio: 0.1,
			},
			want: want{
				poolReward: 0,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "0 value is lost with unequal delegate distribution",
			args: args{
				value:              50,
				numDelegates:       2,
				delegateBal:        []currency.Coin{13, 19},
				serviceChargeRatio: 0.5,
			},
			want: want{
				poolReward: 25,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "100 percent service charge",
			args: args{
				value:              50,
				numDelegates:       2,
				delegateBal:        []currency.Coin{13, 19},
				serviceChargeRatio: 1,
			},
			want: want{
				poolReward: 50,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "no delegates with 0 service charge",
			args: args{
				value:              100,
				numDelegates:       0,
				delegateBal:        []currency.Coin{},
				serviceChargeRatio: 0,
			},
			want: want{
				poolReward: 100,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "single delegates with 1 coin and 50 percent service charge",
			args: args{
				value:              1,
				numDelegates:       1,
				delegateBal:        []currency.Coin{1},
				serviceChargeRatio: 0.5,
			},
			want: want{
				poolReward: 0,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
		{
			name: "single delegates with 100 percent service charge",
			args: args{
				value:              20,
				numDelegates:       1,
				delegateBal:        []currency.Coin{100},
				serviceChargeRatio: 1,
			},
			want: want{
				poolReward: 20,
				eventTags:  []event.EventTag{event.TagStakePoolReward},
				err:        false,
				errMsg:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp, balances := setup(t, tt.args)
			err := sp.DistributeRewardsRandN(tt.args.value, providerID, providerType, RoundRandomSeed, NumMinerDelegatesRewarded, spenum.BlockRewardBlobber, balances)
			require.EqualValues(t, tt.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errMsg, err.Error())
			}
			events := balances.GetEvents()
			require.EqualValues(t, len(tt.want.eventTags), len(events))
			if len(tt.want.eventTags) > 0 {
				for i, event := range events {
					require.EqualValues(t, tt.want.eventTags[i], event.Tag)
				}
			}
			require.EqualValues(t, tt.want.poolReward, sp.Reward)
		})
	}

}

func TestGetOrderedPools(t *testing.T) {
	sp := &StakePool{
		Pools: map[string]*DelegatePool{
			"p1": &DelegatePool{DelegateID: "p1"},
			"p2": &DelegatePool{DelegateID: "p2"},
			"p3": &DelegatePool{DelegateID: "p3"},
		},
	}

	ps := sp.GetOrderedPools()
	require.EqualValues(t, 3, len(ps))
	require.Equal(t, "p1", ps[0].DelegateID)
	require.Equal(t, "p2", ps[1].DelegateID)
	require.Equal(t, "p3", ps[2].DelegateID)
}

func Test_validateLockRequest(t *testing.T) {
	type args struct {
		t   *transaction.Transaction
		sp  AbstractStakePool
		vs  ValidationSettings
		err error
	}
	clientId := "randomHash"
	clientId2 := "randomHash2"
	clientId3 := "randomHash3"
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "first stake should pass",
			args: args{
				t: &transaction.Transaction{
					ClientID: clientId,
					Value:    10,
				},
				sp: &StakePool{
					Pools: map[string]*DelegatePool{
						clientId: {
							Balance: 10,
						},
						clientId2: {
							Balance: 20,
						},
					},
					Settings: Settings{
						ServiceChargeRatio: 0,
						MaxNumDelegates:    2,
					},
				},
				vs: ValidationSettings{
					MinStake:        0,
					MaxStake:        50,
					MaxNumDelegates: 2,
				},
				err: nil,
			},
			want:    "",
			wantErr: false,
		}, {
			name: "second stake should pass",
			args: args{
				t: &transaction.Transaction{
					ClientID: clientId,
					Value:    20,
				},
				sp: &StakePool{
					Pools: map[string]*DelegatePool{
						clientId: {
							Balance: 30,
						},
						clientId2: {
							Balance: 20,
						},
					},
					Settings: Settings{
						ServiceChargeRatio: 0,
						MaxNumDelegates:    2,
					},
				},
				vs: ValidationSettings{
					MinStake:        0,
					MaxStake:        50,
					MaxNumDelegates: 2,
				},
				err: nil,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "first stake should fail on max limit",
			args: args{
				t: &transaction.Transaction{
					ClientID: clientId,
					Value:    60,
				},
				sp: &StakePool{
					Pools: map[string]*DelegatePool{
						clientId: {
							Balance: 0,
						},
						clientId2: {
							Balance: 20,
						},
					},
					Settings: Settings{
						ServiceChargeRatio: 0,
						MaxNumDelegates:    2,
					},
				},
				vs: ValidationSettings{
					MinStake:        0,
					MaxStake:        50,
					MaxNumDelegates: 2,
				},
				err: nil,
			},
			want:    "",
			wantErr: true,
		}, {
			name: "second stake should fail on max limit",
			args: args{
				t: &transaction.Transaction{
					ClientID: clientId,
					Value:    20,
				},
				sp: &StakePool{
					Pools: map[string]*DelegatePool{
						clientId: {
							Balance: 40,
						},
						clientId2: {
							Balance: 20,
						},
					},
					Settings: Settings{
						ServiceChargeRatio: 0,
						MaxNumDelegates:    2,
					},
				},
				vs: ValidationSettings{
					MinStake:        0,
					MaxStake:        50,
					MaxNumDelegates: 2,
				},
				err: nil,
			},
			want:    "",
			wantErr: true,
		}, {
			name: "stake should fail on min limit",
			args: args{
				t: &transaction.Transaction{
					ClientID: clientId,
					Value:    10,
				},
				sp: &StakePool{
					Pools: map[string]*DelegatePool{
						clientId: {
							Balance: 30,
						},
						clientId2: {
							Balance: 20,
						},
					},
					Settings: Settings{
						ServiceChargeRatio: 0,
						MaxNumDelegates:    2,
					},
				},
				vs: ValidationSettings{
					MinStake:        20,
					MaxStake:        50,
					MaxNumDelegates: 2,
				},
				err: nil,
			},
			want:    "",
			wantErr: true,
		}, {
			name: "stake should fail on delegates limit",
			args: args{
				t: &transaction.Transaction{
					ClientID: "clientIdTxn",
					Value:    10,
				},
				sp: &StakePool{
					Pools: map[string]*DelegatePool{
						clientId: {
							Balance: 30,
						},
						clientId2: {
							Balance: 20,
						},
						clientId3: {
							Balance: 20,
						},
					},
					Settings: Settings{
						ServiceChargeRatio: 0,
						MaxNumDelegates:    2,
					},
				},
				vs: ValidationSettings{
					MinStake:        0,
					MaxStake:        50,
					MaxNumDelegates: 2,
				},
				err: nil,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balances := newTestBalances(t, false)

			b := &block.Block{}
			b.Round = 1
			balances.block = b

			got, err := validateLockRequest(tt.args.t, tt.args.sp, tt.args.vs, balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLockRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateLockRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
