package event

import (
	"0chain.net/core/encryption"
	"sort"
)

type RoundEnd struct {
	EventCount int    `json:"round"`
	Hash       []byte `json:"hash"`
}

type EventList []Event

func (el *EventList) AddEvent(event Event) {
	if len(event.Hash) == 0 {
		event.GetHashBytes()
	}
	*el = append(*el, event)
}

func compareHash(eventMap map[string]bool) {

}

func (el *EventList) GetHash() []byte {
	var data []byte
	sort.Slice(el, func(i, j int) bool {
		return (*el)[i].Hash < (*el)[j].Hash
	})
	for _, event := range *el {
		data = append(data, []byte(event.Hash)...)
	}
	return encryption.RawHash(data)
}
