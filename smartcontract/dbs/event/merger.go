package event

import (
	"reflect"
)

type eventMergeMiddleware func([]Event) ([]Event, error)

type eventsMerger interface {
	filter(Event) bool
	merge(round int64, blockHash string) (*Event, error)
}

type eventsMergerImpl[T any] struct {
	tag         EventTag
	events      []Event
	middlewares []eventMergeMiddleware
}

func newEventsMerger[T any](tag EventTag, middlewares ...eventMergeMiddleware) *eventsMergerImpl[T] {
	return &eventsMergerImpl[T]{
		tag:         tag,
		middlewares: append([]eventMergeMiddleware{}, middlewares...),
	}
}

func (em *eventsMergerImpl[T]) filter(event Event) bool {
	if event.Tag == em.tag {
		em.events = append(em.events, event)
		return true
	}

	return false
}

func (em *eventsMergerImpl[T]) merge(round int64, blockHash string) (*Event, error) {
	if len(em.events) == 0 {
		return nil, nil
	}

	events := em.events
	for _, mHandler := range em.middlewares {
		var err error
		events, err = mHandler(events)
		if err != nil {
			return nil, err
		}
	}

	data := make([]T, 0, len(events))
	for _, e := range events {
		if reflect.TypeOf(e.Data).Kind() == reflect.Slice {
			pd, ok := fromEvent[[]T](e.Data)
			if !ok {
				return nil, ErrInvalidEventData
			}

			data = append(data, *pd...)
		} else {
			pd, ok := fromEvent[T](e.Data)
			if !ok {
				return nil, ErrInvalidEventData
			}
			data = append(data, *pd)
		}
	}

	return &Event{
		Type:        TypeStats,
		Tag:         em.tag,
		BlockNumber: round,
		Index:       blockHash,
		Data:        data,
	}, nil
}

// withUniqueEventOverwrite is an event merge middleware that will overwrite the exist
// event with later event that has the same index. It should only be used when
// you are sure that the overwritten would not cause problem.
func withUniqueEventOverwrite() eventMergeMiddleware {
	return func(events []Event) ([]Event, error) {
		eMap := make(map[string]Event, len(events))
		for _, e := range events {
			eMap[e.Index] = e
		}

		ret := make([]Event, 0, len(eMap))
		for _, e := range eMap {
			ret = append(ret, e)
		}

		return ret, nil
	}
}

// mergeEventsFunc merge a and b, data will be merged to a and returned.
type mergeEventsFunc[T any] func(a, b *T) (*T, error)

// withEventMerge merge events that has the same index and merge the
// event data by calling the mergeFunc function.
func withEventMerge[T any](mergeFunc mergeEventsFunc[T]) eventMergeMiddleware {
	return func(events []Event) ([]Event, error) {
		eMap := make(map[string]*Event, len(events))
		for i, e := range events {
			// exist event
			ee, ok := eMap[e.Index]
			if !ok {
				eMap[e.Index] = &events[i]
				continue
			}

			// exist event data
			eeData, ok := fromEvent[T](ee.Data)
			if !ok {
				return nil, ErrInvalidEventData
			}

			eData, ok := fromEvent[T](e.Data)
			if !ok {
				return nil, ErrInvalidEventData
			}

			// new event data
			newData, err := mergeFunc(eeData, eData)
			if err != nil {
				return nil, err
			}

			err = setEventData[T](ee, *newData)
			if err != nil {
				return nil, err
			}
		}

		ret := make([]Event, 0, len(eMap))
		for _, e := range eMap {
			ret = append(ret, *e)
		}

		return ret, nil
	}
}

func mergeAddProviderEvents[T any](tag EventTag, middlewares ...eventMergeMiddleware) eventsMerger {
	return newEventsMerger[T](tag, middlewares...)
}
