package storagesc

import (
	"sort"
)

//go:generate msgp -io=false -tests=false -v

// SortedList represents a unique sorted list of strings for O(logN) access
type SortedList []string

func (sl SortedList) getIndex(id string) (i int, ok bool) {
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

//nolint:golint,unused
func (sl *SortedList) removeByIndex(i int) {
	(*sl) = append((*sl)[:i], (*sl)[i+1:]...)
}

func (sl *SortedList) remove(id string) (ok bool) {
	var i int
	if i, ok = sl.getIndex(id); !ok {
		return // false
	}
	sl.removeByIndex(i)
	return true // removed
}

func (sl *SortedList) add(id string) (ok bool) {
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

// SortedBlobbers represents a unique sorted list of blobbers for O(logN) access
type SortedBlobbers []*StorageNode

//nolint:golint,unused
func (sb SortedBlobbers) getIndex(id string) (i int, ok bool) {
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

func (sb SortedBlobbers) get(id string) (b *StorageNode, ok bool) {
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

//nolint:golint,unused
func (sb *SortedBlobbers) removeByIndex(i int) {
	(*sb) = append((*sb)[:i], (*sb)[i+1:]...)
}

//nolint:golint,unused
func (sb *SortedBlobbers) remove(id string) (ok bool) {
	var i int
	if i, ok = sb.getIndex(id); !ok {
		return // false
	}
	sb.removeByIndex(i)
	return true // removed
}

func (sb *SortedBlobbers) add(b *StorageNode) (ok bool) {
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

// replace if found
//nolint:golint,unused
func (sb *SortedBlobbers) update(b *StorageNode) (ok bool) {
	var i int
	if i, ok = sb.getIndex(b.ID); !ok {
		return
	}
	(*sb)[i] = b // replace
	return
}

func (sb SortedBlobbers) copy() (cp []*StorageNode) {
	cp = make([]*StorageNode, 0, len(sb))
	for _, b := range sb {
		cp = append(cp, b)
	}
	return
}
