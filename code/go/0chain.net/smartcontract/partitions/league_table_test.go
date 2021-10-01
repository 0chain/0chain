package partitions

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	const (
		mockNewId = "mock new id"
	)

	type change struct {
		item     OrderedPartitionItem
		from, to PartitionId
	}
	type callbackCalls []change

	callbacks := callbackCalls{}
	var mockCallBack changePositionHandler = func(
		item OrderedPartitionItem,
		from, to PartitionId,
		_ state.StateContextI,
	) error {
		callbacks = append(callbacks, struct {
			item     OrderedPartitionItem
			from, to PartitionId
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
		newEntry     int64
	}
	type want struct {
		callbacks []change
		error     bool
		errorMsg  string
	}

	var setup = func(t *testing.T, name string, p parameters) args {
		numDivisions := p.numEntries / p.divisionSize
		if p.numEntries%p.divisionSize > 0 {
			numDivisions++
		}

		return args{
			lt: leagueTable{
				Name:         name,
				DivisionSize: p.divisionSize,
				Divisions:    make([]*divison, numDivisions, numDivisions),
				Callback:     mockCallBack,
			},
			in: leagueMember{
				Id:    mockNewId,
				Value: p.newEntry,
			},
			balances: &mocks.StateContextI{},
		}
	}

	setExpectations := func(
		t *testing.T,
		p parameters,
		args args,
		want want,
	) want {
		var lt = mockLeagueTable(
			args.lt.Name,
			p.numEntries,
			args.lt.DivisionSize,
			args.lt.Callback,
		)
		for i := 0; i < len(args.lt.Divisions); i++ {
			args.balances.On(
				"GetTrieNode",
				args.lt.divisionKey(i),
			).Return(lt.Divisions[i], nil).Once()
		}

		want.callbacks = []change{}

		numDivisions := p.numEntries / p.divisionSize
		if p.numEntries%p.divisionSize != 0 {
			numDivisions++
		}
		//lowestFull := 0 == p.numEntries%p.divisionSize
		//highestValue := numDivisions*p.divisionSize

		var entryDivision = int(p.newEntry) / p.divisionSize
		if int(p.newEntry)%p.divisionSize == 1 {
			entryDivision--
		}
		if entryDivision < 0 {
			entryDivision = 0
		}
		entryDivision = numDivisions - entryDivision

		return want
	}

	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_empty",
			parameters: parameters{
				divisionSize: 5,
				numEntries:   0,
				newEntry:     11,
			},
		},
		/*
			{
				name: "ok_end_first_division",
				parameters: parameters{
					divisionSize: 5,
					numEntries:   15,
					newEntry:     11,
				},
			},
			{
				name: "ok_first_division",
				parameters: parameters{
					divisionSize: 5,
					numEntries:   15,
					newEntry:     14,
				},
			},

			{
				name: "ok_last_partition",
				parameters: parameters{
					divisionSize: 5,
					numEntries:   12,
					newEntry:     4,
				},
			},

			{
				name: "ok_new_partition",
				parameters: parameters{
					divisionSize: 5,
					numEntries:   15,
					newEntry:     -3,
				},
			},*/
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := setup(t, tt.name, tt.parameters)
			tt.want = setExpectations(t, tt.parameters, args, tt.want)

			callbacks = callbackCalls{}
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

func TestRemove(t *testing.T) {
	const (
		mockNewId = "mock new id"
	)

	type change struct {
		item     OrderedPartitionItem
		from, to PartitionId
	}
	type callbackCalls []change

	callbacks := callbackCalls{}
	var mockCallBack changePositionHandler = func(
		item OrderedPartitionItem,
		from, to PartitionId,
		_ state.StateContextI,
	) error {
		callbacks = append(callbacks, struct {
			item     OrderedPartitionItem
			from, to PartitionId
		}{
			item: item,
			from: from,
			to:   to,
		})
		return nil
	}

	type args struct {
		lt          leagueTable
		name        string
		partitionId PartitionId
		balances    *mocks.StateContextI
	}
	type parameters struct {
		divisionSize int
		numEntries   int
		name         string
		divisionId   int
	}
	type want struct {
		callbacks []change
		error     bool
		errorMsg  string
	}

	var setup = func(t *testing.T, name string, p parameters) args {
		numDivisions := p.numEntries / p.divisionSize
		if p.numEntries%p.divisionSize > 0 {
			numDivisions++
		}

		return args{
			lt: leagueTable{
				Name:         name,
				DivisionSize: p.divisionSize,
				Divisions:    make([]*divison, numDivisions, numDivisions),
				Callback:     mockCallBack,
			},
			name:        p.name,
			partitionId: PartitionId(p.divisionId),
			balances:    &mocks.StateContextI{},
		}
	}

	setExpectations := func(
		t *testing.T,
		p parameters,
		args args,
		want want,
	) want {
		var lt = mockLeagueTable(
			args.lt.Name,
			p.numEntries,
			args.lt.DivisionSize,
			args.lt.Callback,
		)
		for i := 0; i < len(args.lt.Divisions); i++ {
			args.balances.On(
				"GetTrieNode",
				args.lt.divisionKey(i),
			).Return(lt.Divisions[i], nil).Once()
		}
		return want
	}

	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_end_first_division",
			parameters: parameters{
				divisionSize: 5,
				numEntries:   15,
				name:         "divisions 1 position 4",
				divisionId:   1,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := setup(t, tt.name, tt.parameters)
			tt.want = setExpectations(t, tt.parameters, args, tt.want)

			callbacks = callbackCalls{}
			args.lt.OnChangePosition(mockCallBack)
			err := args.lt.Remove(args.name, args.partitionId, args.balances)

			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}

func mockLeagueTable(
	name string,
	entries, divisionSize int,
	callback changePositionHandler,
) leagueTable {
	var lt = leagueTable{
		Name:         name,
		DivisionSize: divisionSize,
		Callback:     callback,
	}
	fullDivisions := entries / divisionSize
	for i := 0; i < fullDivisions; i++ {
		var nextDivision divison
		for j := 0; j < divisionSize; j++ {
			nextDivision.Members = append(nextDivision.Members, leagueMember{
				Id:    "divisions " + strconv.Itoa(i) + " position " + strconv.Itoa(j),
				Value: int64((fullDivisions+1)*divisionSize - i*divisionSize - j),
			})
		}
		lt.Divisions = append(lt.Divisions, &nextDivision)
	}
	var lastDivision divison
	for i := 0; i < entries%divisionSize; i++ {
		lastDivision.Members = append(lastDivision.Members, leagueMember{
			Id:    "divisions " + strconv.Itoa(fullDivisions+1) + " position " + strconv.Itoa(i),
			Value: int64((fullDivisions+1)*divisionSize - (fullDivisions)*divisionSize - i),
		})
	}
	if len(lastDivision.Members) > 0 {
		lt.Divisions = append(lt.Divisions, &lastDivision)
	}
	return lt
}
