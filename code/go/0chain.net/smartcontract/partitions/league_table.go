package partitions

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"0chain.net/core/util"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

type leagueMember struct {
	Id    string `json:"id"`
	Value int64  `json:"value"`
}

func (lm leagueMember) Name() string {
	return lm.Id
}

func (lm leagueMember) GraterThan(item PartitionItem) bool {
	lm2, ok := item.(leagueMember)
	if !ok {
		panic(fmt.Sprintf("GreaterThan: corrupt league, %v is not a league member", item))
	}
	return lm.Value > lm2.Value
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
const notFound = -1

type divison struct {
	Members []leagueMember `json:"members"`
	changed bool
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

func (d *divison) set(index int, opi OrderedPartitionItem) error {
	lm, ok := opi.(leagueMember)
	if !ok {
		fmt.Errorf("%v is not a league member", opi)
	}
	if len(d.Members) > index {
		d.Members[index] = lm
	} else if len(d.Members) == index {
		d.Members = append(d.Members, lm)
	} else {
		return fmt.Errorf("index %d exceeds division length %d", index, len(d.Members))
	}
	return nil
}

func (d *divison) floor() OrderedPartitionItem {
	return d.Members[len(d.Members)-1]
}

func (d *divison) find(name string) int {
	for i, member := range d.Members {
		if name == member.Name() {
			return i
		}
	}
	return notFound
}

func (d *divison) add(
	in OrderedPartitionItem,
	maxSize int,
	balances state.StateContextI,
) (OrderedPartitionItem, error) {
	index := sort.Search(len(d.Members), func(i int) bool {
		return !d.Members[i].GraterThan(in)
	})
	var relegated OrderedPartitionItem
	if len(d.Members) >= maxSize {
		relegated = d.Members[len(d.Members)-1]
	}

	if index != len(d.Members) {
		d.Members = append(d.Members[:index+1], d.Members[index:len(d.Members)-1]...)
	}
	if err := d.set(index, in); err != nil {
		return nil, err
	}
	d.changed = true
	return relegated, nil
}

//------------------------------------------------------------------------------

type leagueTable struct {
	Name         string                `json:"name"`
	DivisionSize int                   `json:"division_size"`
	Divisions    []*divison            `json:"divisions"`
	Callback     changePositionHandler `json:"on_change_division"`
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
	lt.Callback = f
}

func (lt *leagueTable) Add(in OrderedPartitionItem, balances state.StateContextI) error {
	const notFound = -1
	var targetDivision = notFound
	for i := 0; i < len(lt.Divisions); i++ {
		div, err := lt.getDivision(i, balances)
		if err != nil {
			return err
		}
		if div != nil && (len(div.Members) == 0 || in.GraterThan(div.floor())) {
			targetDivision = i
			break
		}
	}
	if targetDivision == notFound {
		if len(lt.Divisions) == 0 || len(lt.Divisions[len(lt.Divisions)-1].Members) == lt.DivisionSize {
			lt.addDivision()
		}
		targetDivision = len(lt.Divisions) - 1
	}

	relegated, err := lt.Divisions[targetDivision].add(in, lt.DivisionSize, balances)
	if err != nil {
		return err
	}
	lt.Divisions[targetDivision].changed = true
	if lt.Callback != nil {
		if err := lt.Callback(in, NoPartition, PartitionId(targetDivision), balances); err != nil {
			return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
				in, nil, targetDivision)
		}
	}
	lt.Divisions[targetDivision].changed = true

	if relegated != nil {
		err := lt.relegatedTo(relegated, targetDivision+1, balances)
		if err != nil {
			return err
		}
		if err := lt.Callback(relegated, 0, 1, balances); err != nil {
			return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
				relegated, 0, 1)
		}
	}
	return nil
}

func (lt *leagueTable) Remove(name string, index PartitionId, balances state.StateContextI) error {
	div, err := lt.getDivision(int(index), balances)
	if err != nil {
		return err
	}
	if div == nil {
		return fmt.Errorf("partition %v not found", index)
	}

	position := div.find(name)
	if position == notFound {
		return err
	}

	removed := div.Members[position]
	div.Members = append(div.Members[:position], div.Members[position+1:]...)
	div.changed = true
	if lt.Callback != nil {
		if err := lt.Callback(removed, index, NoPartition, balances); err != nil {
			return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
				index, NoPartition, nil)
		}
	}

	if len(div.Members) == lt.DivisionSize-1 {
		promoted, err := lt.promoteFrom(int(index+1), balances)
		if err != nil {
			return err
		}
		if promoted != nil {
			if err := div.set(lt.DivisionSize-1, promoted); err != nil {
				return err
			}
			if lt.Callback != nil {
				if err := lt.Callback(
					promoted, PartitionId(index+1), PartitionId(index), balances,
				); err != nil {
					return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
						promoted, index+1, index)
				}
			}
		} else {
			div.Members = div.Members[:lt.DivisionSize-1]
		}
	}
	if len(div.Members) == 0 {
		lt.Divisions = lt.Divisions[:len(lt.Divisions)-1]
	}
	return nil
}

func (lt *leagueTable) Change(
	changed OrderedPartitionItem,
	from PartitionId,
	balances state.StateContextI,
) error {
	err := lt.Remove(changed.Name(), from, balances)
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
	if div == nil || len(div.Members) == 0 {
		return nil, nil
	}
	var promoted = div.Members[0]
	div.Members = div.Members[1:len(div.Members)]

	div.changed = true

	if len(div.Members) == lt.DivisionSize-1 {
		promotedUp, err := lt.promoteFrom(index+1, balances)
		if err != nil {
			return nil, err
		}
		if promotedUp != nil {
			div.set(lt.DivisionSize-1, promotedUp)
			if lt.Callback != nil {
				if err := lt.Callback(
					promoted, PartitionId(index+1), PartitionId(index), balances,
				); err != nil {
					return nil, fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
						promoted, index+1, index)
				}
			}
		}
	}

	if len(div.Members) == 0 {
		lt.Divisions = lt.Divisions[:len(lt.Divisions)-1]
	}
	return promoted, nil
}

func (lt *leagueTable) addDivision() *divison {
	var newDiv divison
	lt.Divisions = append(lt.Divisions, &newDiv)
	return &newDiv
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
	if div == nil {
		div = lt.addDivision()
	}

	if len(div.Members) == lt.DivisionSize {
		relegated := div.Members[lt.DivisionSize-1]
		err := lt.relegatedTo(relegated, index+1, balances)
		if err != nil {
			return nil
		}
		if lt.Callback != nil {
			if err := lt.Callback(
				relegated, PartitionId(index), PartitionId(index+1), balances,
			); err != nil {
				return fmt.Errorf("running callback, {in: %v, old position: %v, new poslitin: %v}",
					relegated, leaguePosition{index, lt.DivisionSize},
					leaguePosition{index + 1, 0})
			}
		}
		div.Members = append([]leagueMember{{}}, div.Members[:lt.DivisionSize-1]...)
	} else {
		div.Members = append([]leagueMember{{}}, div.Members[:]...)
	}

	if err := div.set(0, in); err != nil {
		return err
	}
	div.changed = true
	return nil
}

func (lt *leagueTable) newDivision() *divison {
	return &divison{
		Members: make([]leagueMember, 0, lt.DivisionSize),
	}
}

func (lt *leagueTable) getDivision(i int, balances state.StateContextI) (*divison, error) {
	if i == len(lt.Divisions) {
		return nil, nil
	}
	if i > len(lt.Divisions) {
		return nil, fmt.Errorf("partition id %v grater than numbr of partitions %v", i, len(lt.Divisions))
	}
	if lt.Divisions[i] != nil {
		return lt.Divisions[i], nil
	}
	var div divison
	val, err := balances.GetTrieNode(lt.divisionKey(i))
	if err != nil {
		return nil, err
	}
	if err := div.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	lt.Divisions[i] = &div
	return &div, nil
}

func (lt *leagueTable) save(balances state.StateContextI) error {
	var numDivisions = 0
	for i, division := range lt.Divisions {
		if division.changed {
			if len(division.Members) > 0 {
				_, err := balances.InsertTrieNode(lt.divisionKey(i), division)
				if err != nil {
					return err
				}
				numDivisions++
			} else {
				_, err := balances.DeleteTrieNode(lt.divisionKey(i))
				if err != nil {
					if err != util.ErrValueNotPresent {
						return err
					}
				}
			}
		}
	}

	lt.Divisions = make([]*divison, numDivisions, numDivisions)
	_, err := balances.InsertTrieNode(lt.Name, lt)
	if err != nil {
		return err
	}

	return nil
}

func GetLeagueTable(
	key datastore.Key,
	balances state.StateContextI,
) (LeagueTable, error) {
	var lt leagueTable
	val, err := balances.GetTrieNode(key)
	if err != nil {
		return nil, err

	}
	if err := lt.Decode(val.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return &lt, nil
}
