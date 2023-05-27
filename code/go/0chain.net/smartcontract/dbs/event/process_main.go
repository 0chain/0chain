//go:build !integration_tests
// +build !integration_tests

package event

func (edb *EventDb) addStat(event Event) (err error) {
	return edb.addStatMain(event)
}
