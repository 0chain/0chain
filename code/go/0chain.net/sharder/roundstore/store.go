package roundstore

/*RoundStore - an interface to read and write round number to storage */
type RoundStore interface {
	Write(roundNum int64) error
	Read() (int64, error)
}

var Store RoundStore

/*GetStore - get the round store that's is setup */
func GetStore() RoundStore {
	return Store
}

/*SetupStore - Setup a file system based round storage */
func SetupStore(store RoundStore) {
	Store = store
}
