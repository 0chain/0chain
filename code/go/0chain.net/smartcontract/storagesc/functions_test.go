package storagesc

import (
	"0chain.net/chaincore/chain/mocks"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestTransferReward(t *testing.T) {
	type parameters struct {
		from         string
		to           string
		value        float64
		delegates    []float64
		seviceCharge float64
	}

	type args struct {
		sscKey   string
		zcnPool  tokenpool.ZcnPool
		sp       *stakePool
		value    state.Balance
		balances cstate.StateContextI
	}

	var setExpectations = func(t *testing.T, p parameters) cstate.StateContextI {
		var balances = &mocks.StateContextI{}
		var serviceCharge = p.value * p.seviceCharge
		balances.On("AddTransfer", &state.Transfer{
			ClientID:   datastore.Key(p.from),
			ToClientID: datastore.Key(p.to),
			Amount:     zcnToBalance(serviceCharge),
		}).Return(nil).Once()

		var total float64
		for _, d := range p.delegates {
			total += d
		}
		var paidToDelegates = p.value - serviceCharge
		for i, d := range p.delegates {
			balances.On("AddTransfer", &state.Transfer{
				ClientID:   datastore.Key(p.from),
				ToClientID: datastore.Key(strconv.Itoa(i)),
				Amount:     zcnToBalance(paidToDelegates * d / total),
			}).Return(nil).Once()
		}
		return balances
	}

	var setup = func(t *testing.T, p parameters) args {
		var zcnPool tokenpool.ZcnPool
		zcnPool.Balance = zcnToBalance(2 * p.value)
		sPool := newStakePool()

		sPool.Settings = stakePoolSettings{
			ServiceCharge:  p.seviceCharge,
			DelegateWallet: p.to,
		}
		for i, d := range p.delegates {
			id := strconv.Itoa(i)
			dPool := delegatePool{}
			dPool.Balance = zcnToBalance(d)
			dPool.DelegateID = id
			sPool.Pools[id] = &dPool
		}

		balances := setExpectations(t, p)

		return args{
			sscKey:   p.from,
			zcnPool:  zcnPool,
			sp:       sPool,
			value:    zcnToBalance(p.value),
			balances: balances,
		}
	}

	type want struct {
		moved    state.Balance
		error    bool
		errorMsg string
	}

	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok",
			parameters: parameters{
				from:         "fred",
				to:           "tommy",
				value:        100,
				delegates:    []float64{1, 5, 2, 1},
				seviceCharge: 0.1,
			},
			want: want{
				moved: 100,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := setup(t, tt.parameters)
			moved, err := transferReward(args.sscKey, args.zcnPool, args.sp, args.value, args.balances)
			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.EqualValues(t, args.value, moved)
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}

func TestMintReward(t *testing.T) {
	type parameters struct {
		to           string
		value        float64
		delegates    []float64
		seviceCharge float64
	}

	type args struct {
		sscKey   string
		zcnPool  tokenpool.ZcnPool
		sp       *stakePool
		value    float64
		balances cstate.StateContextI
	}

	var setExpectations = func(t *testing.T, p parameters) cstate.StateContextI {
		var balances = &mocks.StateContextI{}
		var serviceCharge = p.value * p.seviceCharge
		balances.On("AddMint", &state.Mint{
			Minter:     ADDRESS,
			ToClientID: datastore.Key(p.to),
			Amount:     zcnToBalance(serviceCharge),
		}).Return(nil).Once()

		var total float64
		for _, d := range p.delegates {
			total += d
		}
		var paidToDelegates = p.value - serviceCharge
		for i, d := range p.delegates {
			balances.On("AddMint", &state.Mint{
				Minter:     ADDRESS,
				ToClientID: datastore.Key(strconv.Itoa(i)),
				Amount:     zcnToBalance(paidToDelegates * d / total),
			}).Return(nil).Once()
		}
		return balances
	}

	var setup = func(t *testing.T, p parameters) args {
		sPool := newStakePool()
		sPool.Settings = stakePoolSettings{
			ServiceCharge:  p.seviceCharge,
			DelegateWallet: p.to,
		}
		for i, d := range p.delegates {
			id := strconv.Itoa(i)
			dPool := delegatePool{}
			dPool.Balance = zcnToBalance(d)
			dPool.DelegateID = id
			sPool.Pools[id] = &dPool
		}

		balances := setExpectations(t, p)

		return args{
			sp:       sPool,
			value:    float64(zcnToBalance(p.value)),
			balances: balances,
		}
	}

	type want struct {
		error    bool
		errorMsg string
	}

	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok",
			parameters: parameters{
				to:           "tommy",
				value:        100.0,
				delegates:    []float64{1, 5, 2, 1},
				seviceCharge: 0.1,
			},
		},
		{
			name: "no delegates",
			parameters: parameters{
				to:           "tommy",
				value:        100.0,
				seviceCharge: 0.1,
			},
			want: want{
				error:    true,
				errorMsg: "no stake pools to move tokens to",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := setup(t, tt.parameters)
			err := mintReward(args.sp, args.value, args.balances)
			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
