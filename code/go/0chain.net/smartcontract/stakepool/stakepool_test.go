package stakepool

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/transaction"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
)

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
		err             bool
		errMsg          string
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
				err:             false,
				errMsg:          "",
			},
		},
		{
			name: "100% service charge",
			args: args{
				value:              50,
				numDelegates:       2,
				delegateBal:        []currency.Coin{13, 19},
				serviceChargeRatio: 1,
			},
			want: want{
				poolReward:      50,
				delegateRewards: []currency.Coin{0, 0},
				err:             false,
				errMsg:          "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp, balances := setup(t, tt.args)
			err := sp.DistributeRewards(tt.args.value, providerID, providerType, spenum.BlockRewardBlobber, balances)
			require.EqualValues(t, tt.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errMsg, err.Error())
			}
			validate(t, sp, tt.want)
		})
	}
}

func Test_validateLockRequest(t *testing.T) {
	type args struct {
		t   *transaction.Transaction
		sp  AbstractStakePool
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
						MinStake:           0,
						MaxStake:           50,
						MaxNumDelegates:    2,
						ServiceChargeRatio: 0,
					},
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
						MinStake:           0,
						MaxStake:           50,
						MaxNumDelegates:    2,
						ServiceChargeRatio: 0,
					},
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
						MinStake:           0,
						MaxStake:           50,
						MaxNumDelegates:    2,
						ServiceChargeRatio: 0,
					},
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
						MinStake:           0,
						MaxStake:           50,
						MaxNumDelegates:    2,
						ServiceChargeRatio: 0,
					},
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
						MinStake:           20,
						MaxStake:           50,
						MaxNumDelegates:    2,
						ServiceChargeRatio: 0,
					},
				},
				err: nil,
			},
			want:    "",
			wantErr: true,
		}, {
			name: "stake should fail on delegates limit",
			args: args{
				t: &transaction.Transaction{
					ClientID: clientId3,
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
						MinStake:           0,
						MaxStake:           50,
						MaxNumDelegates:    2,
						ServiceChargeRatio: 0,
					},
				},
				err: nil,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateLockRequest(tt.args.t, tt.args.sp, tt.args.err)
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
