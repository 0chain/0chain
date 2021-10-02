package partitions

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"testing"

	"0chain.net/core/util"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFuzzyLeagueTable(t *testing.T) {
	t.Skip()
	const (
		mockName         = "fuzzy league table"
		mockSeed         = 0
		fuzzyRunLength   = 100
		mockDivisionSize = 10
		addRatio         = 60
		changeRation     = 20
		removeRatio      = 20
	)

	type Action int
	const (
		Add Action = iota
		Remove
		Change
	)

	type methodCall struct {
		action     Action
		item       OrderedPartitionItem
		divisionId PartitionId
	}

	type fuzzyItem struct {
		item     leagueMember
		division PartitionId
	}
	items := []fuzzyItem{}

	getAction := func(testId int) methodCall {
		const totalRation = addRatio + changeRation + removeRatio
		var random int
		if len(items) > 0 {
			random = rand.Intn(totalRation)
		}

		var action methodCall
		switch {
		case random < addRatio:
			action = methodCall{
				action: Add,
				item: leagueMember{
					Id:    "test " + strconv.Itoa(testId),
					Value: int64(rand.Intn(fuzzyRunLength)),
				},
			}
		case random < addRatio+removeRatio:
			toRemove := items[rand.Intn(len(items))]
			action = methodCall{
				action:     Remove,
				item:       toRemove.item,
				divisionId: toRemove.division,
			}
		default:
			toChange := items[rand.Intn(len(items))]
			toChange.item.Value = int64(rand.Intn(fuzzyRunLength))
			action = methodCall{
				action:     Change,
				item:       toChange.item,
				divisionId: toChange.division,
			}
		}
		return action
	}

	var mockCallBack changePositionHandler = func(
		item OrderedPartitionItem,
		from, to PartitionId,
		_ state.StateContextI,
	) error {
		fmt.Println("\tcallback item", item, "from", from, "to", to)
		if from == NoPartition {
			items = append(items, fuzzyItem{
				item:     item.(leagueMember),
				division: to,
			})
			return nil
		}

		for i := 0; i < len(items); i++ {
			if items[i].item.Id == item.Name() {
				if items[i].division != from {
					require.EqualValues(t, items[i].division, from)
				}
				if to == NoPartition {
					items = append(items[:i], items[i+1:]...)
				} else {
					items[i].division = to
				}
				return nil
			}
		}

		return fmt.Errorf("not found %v, %d, %d", item, from, to)
	}

	var (
		balances = &mocks.StateContextI{}
		lt       = leagueTable{
			Name:         mockName,
			DivisionSize: mockDivisionSize,
			Callback:     mockCallBack,
		}
	)
	for i := 0; i < len(lt.Divisions); i++ {
		balances.On(
			"GetTrieNode",
			lt.divisionKey(i),
		).Return(nil, util.ErrValueNotPresent).Once()
	}

	rand.Seed(mockSeed)
	var actions []methodCall
	for i := 0; i < fuzzyRunLength; i++ {
		action := getAction(i)
		actions = append(actions, action)
		fmt.Println("i", i, "action", action)
		switch action.action {
		case Add:
			err := lt.Add(action.item, balances)
			require.NoError(t, err, fmt.Sprintf("action Add: %v, error: %v", action, err))
		case Remove:
			err := lt.Remove(action.item.Name(), action.divisionId, balances)
			require.NoError(t, err, fmt.Sprintf("action Remove: %v, error: %v", action, err))
		case Change:
			err := lt.Change(action.item, action.divisionId, balances)
			require.NoError(t, err, fmt.Sprintf("action Change: %v, error: %v", action, err))
		default:
			require.Fail(t, "action not found")
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].item.Value < items[j].item.Value
	})
	itemIndex := 0
	for i := 0; i < len(lt.Divisions); i++ {
		for j := 0; j < len(lt.Divisions[i].Members); j++ {
			require.EqualValues(
				t,
				items[itemIndex].item.Value,
				lt.Divisions[i].Members[j].Value,
			)
			itemIndex++
		}
	}
	require.EqualValues(t, itemIndex, len(items))
}

func TestAdd(t *testing.T) {
	t.Skip()
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
	t.Skip()
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
			).Return(lt.Divisions[i], nil).Maybe()
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
		{
			name: "ok_last_item",
			parameters: parameters{
				divisionSize: 5,
				numEntries:   1,
				name:         "divisions 0 position 0",
				divisionId:   0,
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

func TestChange(t *testing.T) {
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
		item     OrderedPartitionItem
		id       PartitionId
		balances *mocks.StateContextI
	}
	type parameters struct {
		divisionSize int
		divisionId   int
		numEntries   int
		name         string
		value        int64
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
			item: leagueMember{
				Id:    p.name,
				Value: p.value,
			},
			id:       PartitionId(p.divisionId),
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
			).Return(lt.Divisions[i], nil).Maybe()
		}
		return want
	}

	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_up",
			parameters: parameters{
				divisionSize: 5,
				numEntries:   20,
				name:         "divisions 2 position 4",
				divisionId:   2,
				value:        700,
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
			err := args.lt.Change(args.item, args.id, args.balances)

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
