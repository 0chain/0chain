package event

import "0chain.net/chaincore/config"

type Settings struct {
	Debug           bool
	AggregatePeriod int64
	PageLimit       int64
}

func extractSettings(config config.DbAccess) Settings {
	return Settings{
		Debug:           config.Debug,
		AggregatePeriod: config.AggregatePeriod,
		PageLimit:       config.PageLimit,
	}
}

func (edb *EventDb) updateSettings(config config.DbAccess) {
	edb.settings = extractSettings(config)
}

func (edb *EventDb) AggregatePeriod() int64 {
	return edb.settings.AggregatePeriod
}

func (edb *EventDb) PageLimit() int64 {
	return edb.settings.PageLimit
}
