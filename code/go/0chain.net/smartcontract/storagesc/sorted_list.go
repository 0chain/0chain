package storagesc

import (
	"sort"
)

// a unique sorted list of strings for O(logN) access
type sortedList []string

func (sl sortedList) getIndex(id string) (i int, ok bool) {

	i = sort.Search(len(sl), func(i int) bool {
		return sl[i] >= id
	})

	if i == len(sl) {
		return // not found
	}

	if sl[i] == id {
		return i, true // found
	}

	return // not found
}

func (sl *sortedList) removeByIndex(i int) {
	(*sl) = append((*sl)[:i], (*sl)[i+1:]...)
}

func (sl *sortedList) remove(id string) (ok bool) {
	var i int
	if i, ok = sl.getIndex(id); !ok {
		return // false
	}
	sl.removeByIndex(i)
	return true // removed
}

func (sl *sortedList) add(id string) (ok bool) {

	if len(*sl) == 0 {
		(*sl) = append((*sl), id)
		return true // added
	}

	var i = sort.Search(len(*sl), func(i int) bool {
		return (*sl)[i] >= id
	})

	// out of bounds
	if i == len(*sl) {
		(*sl) = append((*sl), id)
		return true // added
	}

	// the same
	if (*sl)[i] == id {
		return false // already have
	}

	// next
	(*sl) = append((*sl)[:i], append([]string{id}, (*sl)[i:]...)...)
	return true // added
}
