package partitions

import (
	"testing"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	const ()

	var ()

	callbacks := []struct {
		item     OrderedPartitionItem
		from, to PartitionLocation
	}{}
	var mockCallBack changePositionHandler = func(
		item OrderedPartitionItem,
		from, to PartitionLocation,
		_ state.StateContextI,
	) error {
		callbacks = append(callbacks, struct {
			item     OrderedPartitionItem
			from, to PartitionLocation
		}{
			item: item,
			from: from,
			to:   to,
		})
		return nil
	}

	type args struct {
		lt       leagueTable
		in       OrderedPartitionItem
		balances *mocks.StateContextI
	}
	type parameters struct {
		divisionSize int
		numEntries   int
	}
	type want struct {
		error    bool
		errorMsg string
	}

	var setup = func(t *testing.T, p parameters) args {
		return args{}
	}

	setExpectations := func(
		t *testing.T,
		p parameters,
		balances *mocks.StateContextI,
	) {

	}

	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name:       "ok",
			parameters: parameters{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := setup(t, tt.parameters)
			setExpectations(t, tt.parameters, args.balances)

			args.lt.OnChangePosition(mockCallBack)
			err := args.lt.Add(args.in, args.balances)

			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
