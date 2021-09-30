package partitions

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type leagueMember struct {
	id    string
	value int64
}

func (lm leagueMember) Name() string {
	return lm.id
}

func (lm leagueMember) GraterThan(lm2 leagueMember) bool {
	return lm.value > lm2.value
}

//------------------------------------------------------------------------------

type leaguePosition struct {
	division int
	position int
}

func (lp leaguePosition) PartitionId() int {
	return lp.division
}

func (lp leaguePosition) Position() int {
	return lp.position
}

//------------------------------------------------------------------------------

type divison struct {
	//MaxLength int                    `json:"max_length"`
	Members []OrderedPartitionItem `json:"members"`
	Changed bool
}

func (d *divison) Encode() []byte {
	var b, err = json.Marshal(d)
	if err != nil {
		panic(err)
	}
	return b
}

func (d *divison) Decode(b []byte) error {
	return json.Unmarshal(b, d)
}

func (d *divison) floor() OrderedPartitionItem {
	return d.Members[len(d.Members)-1]
}

func (d *divison) add(
	in OrderedPartitionItem,
	maxSize int,
	balances state.StateContextI,
) (int, OrderedPartitionItem) {
	index := sort.Search(len(d.Members), func(i int) bool {
		return !d.Members[i].GraterThan(in)
	})
	var relegated OrderedPartitionItem
	if len(d.Members) >= maxSize {
		relegated = d.Members[len(d.Members)-1]
	}
	d.Members = append(d.Members[:index+1], d.Members[index:len(d.Members)-1]...)
	d.Members[index] = in
	d.Changed = true
	return index, relegated
}

//------------------------------------------------------------------------------

type leagueTable struct {
	Name         string `json:"name"`
	DivisionSize int    `json:"division_size"`
	Divisions    []divison
	callback     changePositionHandler `json:"on_change_division"`
}

func (lt *leagueTable) divisionKey(index int) datastore.Key {
	return datastore.Key(lt.Name + encryption.Hash(":division:"+strconv.Itoa(index)))
}

func (lt *leagueTable) Encode() []byte {
	var b, err = json.Marshal(lt)
	if err != nil {
		panic(err)
	}
	return b
}

func (lt *leagueTable) Decode(b []byte) error {
	return json.Unmarshal(b, lt)
}

func (lt *leagueTable) OnChangePosition(f changePositionHandler) {
	lt.callback = f
}

func (lt *leagueTable) Add(in OrderedPartitionItem, balances state.StateContextI) error {
	var targetDivision = len(lt.Divisions)
	for i := 0; i < len(lt.Divisions); i++ {
		div, err := lt.getDivision(i, balances)
		if err != nil {
			return err
		}
		if len(div.Members) == 0 || in.GraterThan(div.floor()) {
			targetDivision = i
			break
		}
	}

	index, relegated := lt.Divisions[targetDivision].add(in, lt.DivisionSize, balances)
	lt.Divisions[targetDivision].Changed = true
	insertPosition := leaguePosition{
		division: targetDivision,
		position: index,
	}
	if err := lt.callback(in, nil, insertPosition, balances); err != nil {
		return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
			in, nil, insertPosition)
	}

	lt.Divisions[targetDivision].Changed = true
	if relegated != nil {
		err := lt.relegatedTo(relegated, targetDivision+1, balances)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lt *leagueTable) Remove(place PartitionLocation, balances state.StateContextI) error {
	div, err := lt.getDivision(place.PartitionId(), balances)
	if err != nil {
		return err
	}
	removed := div.Members[place.Position()]
	div.Members = append(div.Members[:place.Position()], div.Members[place.Position()+1:]...)
	div.Changed = true
	if err := lt.callback(removed, place, nil, balances); err != nil {
		return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
			removed, place, nil)
	}

	if len(div.Members) == lt.DivisionSize-1 {
		promoted, err := lt.promoteFrom(place.PartitionId()+1, balances)
		if err != nil {
			return err
		}
		div.Members[lt.DivisionSize] = promoted
	}
	return nil
}

func (lt *leagueTable) Change(
	changed OrderedPartitionItem,
	at PartitionLocation,
	balances state.StateContextI,
) error {
	err := lt.Remove(at, balances)
	if err != nil {
		return err
	}
	return lt.Add(changed, balances)
}

func (lt *leagueTable) promoteFrom(
	index int,
	balances state.StateContextI,
) (OrderedPartitionItem, error) {
	div, err := lt.getDivision(index, balances)
	if err != nil {
		return nil, err
	}
	if len(div.Members) == 0 {
		return nil, nil
	}
	var promoted = div.Members[0]
	div.Members = div.Members[1:len(div.Members)]
	div.Changed = true
	if err := lt.callback(
		promoted,
		leaguePosition{index, 0},
		leaguePosition{index - 1, lt.DivisionSize},
		balances,
	); err != nil {
		return nil, fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
			promoted, leaguePosition{index, 0},
			leaguePosition{index - 1, lt.DivisionSize})
	}

	if len(div.Members) == lt.DivisionSize-1 {
		promotedUp, err := lt.promoteFrom(index+1, balances)
		if err != nil {
			return nil, err
		}
		if promotedUp != nil {
			div.Members[lt.DivisionSize-1] = promotedUp
		}
	}
	return promoted, nil
}

func (lt *leagueTable) relegatedTo(
	in OrderedPartitionItem,
	index int,
	balances state.StateContextI,
) error {
	div, err := lt.getDivision(index, balances)
	if err != nil {
		return err
	}

	var relegated OrderedPartitionItem
	if len(div.Members) == lt.DivisionSize {
		relegated = div.Members[lt.DivisionSize]
		if err := lt.callback(
			relegated,
			leaguePosition{index, lt.DivisionSize},
			leaguePosition{index + 1, 0},
			balances,
		); err != nil {
			return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
				relegated, leaguePosition{index, lt.DivisionSize},
				leaguePosition{index + 1, 0})
		}
	}
	div.Members = append([]OrderedPartitionItem{in}, div.Members[:lt.DivisionSize-1]...)
	div.Changed = true

	if relegated != nil {
		err := lt.relegatedTo(relegated, index+1, balances)
		if err != nil {
			return nil
		}
	}

	return nil
}

func (lt *leagueTable) newDivision() *divison {
	return &divison{
		Members: make([]OrderedPartitionItem, 0, lt.DivisionSize),
	}
}

func (lt *leagueTable) getDivision(i int, balances state.StateContextI) (*divison, error) {
	if len(lt.Divisions[i].Members) > 0 {
		return &lt.Divisions[i], nil
	}
	var div *divison
	val, err := balances.GetTrieNode(lt.divisionKey(i))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return lt.newDivision(), nil
	}
	if err := div.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return div, nil
}

func (lt *leagueTable) save(balances state.StateContextI) error {
	_, err := balances.InsertTrieNode(lt.Name, lt)
	if err != nil {
		return err
	}
	for i, division := range lt.Divisions {
		if division.Changed {
			_, err := balances.InsertTrieNode(lt.divisionKey(i), &division)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func GetLeagueTable(
	key datastore.Key,
	balances state.StateContextI,
) (LeagueTable, error) {
	var lt *leagueTable
	val, err := balances.GetTrieNode(key)
	if err != nil {
		return nil, err

	}
	if err := lt.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return lt, nil
}
