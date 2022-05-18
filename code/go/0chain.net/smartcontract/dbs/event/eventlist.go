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

func (el *EventList) GetHash() []byte {
	var hashes []string
	for _, event := range *el {
		hashes = append(hashes, event.Hash)
	}
	sort.Strings(hashes)

	var data []byte
	for _, hash := range hashes {
		data = append(data, []byte(hash)...)
	}
	return encryption.RawHash(data)
}
