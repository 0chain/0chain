package event

type Error struct {
	TransactionID string
	Error         string
}

func (edb *EventDb) addError(err Error) error {
	return edb.Store.Get().Create(&err).Error
}
