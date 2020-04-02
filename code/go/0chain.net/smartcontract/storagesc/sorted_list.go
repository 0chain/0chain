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

// sorted blobbers

// a unique sorted list of blobbers for O(logN) access
type sortedBlobbers []*StorageNode

func (sb sortedBlobbers) getIndex(id string) (i int, ok bool) {
	i = sort.Search(len(sb), func(i int) bool {
		return sb[i].ID >= id
	})
	if i == len(sb) {
		return // not found
	}
	if sb[i].ID == id {
		return i, true // found
	}
	return // not found
}

func (sb sortedBlobbers) get(id string) (b *StorageNode, ok bool) {
	var i = sort.Search(len(sb), func(i int) bool {
		return sb[i].ID >= id
	})
	if i == len(sb) {
		return // not found
	}
	if sb[i].ID == id {
		return sb[i], true // found
	}
	return // not found
}

func (sb *sortedBlobbers) removeByIndex(i int) {
	(*sb) = append((*sb)[:i], (*sb)[i+1:]...)
}

func (sb *sortedBlobbers) remove(id string) (ok bool) {
	var i int
	if i, ok = sb.getIndex(id); !ok {
		return // false
	}
	sb.removeByIndex(i)
	return true // removed
}

func (sb *sortedBlobbers) add(b *StorageNode) (ok bool) {
	if len(*sb) == 0 {
		(*sb) = append((*sb), b)
		return true // added
	}
	var i = sort.Search(len(*sb), func(i int) bool {
		return (*sb)[i].ID >= b.ID
	})
	// out of bounds
	if i == len(*sb) {
		(*sb) = append((*sb), b)
		return true // added
	}
	// the same
	if (*sb)[i].ID == b.ID {
		(*sb)[i] = b // replace
		return false // already have
	}
	// next
	(*sb) = append((*sb)[:i], append([]*StorageNode{b}, (*sb)[i:]...)...)
	return true // added
}
