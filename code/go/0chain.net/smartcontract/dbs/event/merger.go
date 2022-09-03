package event

type eventMergeMiddleware func([]Event) ([]Event, error)

type eventsMerger interface {
	filter(Event) bool
	merge(round int64, blockHash string) (*Event, error)
}

type eventsMergerImpl[T any] struct {
	tag         int
	events      []Event
	middlewares []eventMergeMiddleware
}

func newEventsMerger[T any](tag EventTag, middlewares ...eventMergeMiddleware) *eventsMergerImpl[T] {
	return &eventsMergerImpl[T]{
		tag:         int(tag),
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
		pd, ok := fromEvent[T](e.Data)
		if !ok {
			return nil, ErrInvalidEventData
		}
		data = append(data, *pd)
	}

	return &Event{
		Type:        int(TypeStats),
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

// addEventsValueFunc add a and b
type addEventsValueFunc[T any] func(a, b *T) (*T, error)

// withUniqueEventAdded merge events that has the same index and add up the
// event value by calling the addFunc function.
func withUniqueEventAdded[T any](addFunc addEventsValueFunc[T]) eventMergeMiddleware {
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

			// new event data after adding
			newData, err := addFunc(eeData, eData)
			if err != nil {
				return nil, err
			}

			ee.Data = newData
		}

		ret := make([]Event, 0, len(eMap))
		for _, e := range eMap {
			ret = append(ret, *e)
		}

		return ret, nil
	}
}

func newUserEventsMerger() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagAddOrOverwriteUser, withUniqueEventOverwrite())
}

func newTransactionsEventsMerger() *eventsMergerImpl[Transaction] {
	return newEventsMerger[Transaction](TagAddTransactions)
}

func newBlobberTotalStakesEventsMerger() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalStake, withBlobberTotalStakesAdded())
}

func newBlobberTotalOffersEventsMerger() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalOffers, withBlobberTotalOffersAdded())
}

func withBlobberTotalStakesAdded() eventMergeMiddleware {
	return withUniqueEventAdded(func(a, b *Blobber) (*Blobber, error) {
		a.TotalStake += b.TotalStake
		return a, nil
	})
}

func withBlobberTotalOffersAdded() eventMergeMiddleware {
	return withUniqueEventAdded(func(a, b *Blobber) (*Blobber, error) {
		a.OffersTotal += b.OffersTotal
		return a, nil
	})
}
