package versions

import (
	"sync"

	"0chain.net/core/util"
	"github.com/blang/semver/v4"
)

var (
	emptyVersion semver.Version

	// registered versions
	versions               []*Version
	isStateReadyCheckFuncs []isStateReadyCheckFunc

	initOnce sync.Once
)

// IsVersionsEmpty checks if any version is empty
func IsVersionsEmpty() bool {
	if len(versions) == 0 {
		return true
	}

	for _, v := range versions {
		if v.IsEmpty() {
			return true
		}
	}

	return false
}

// InitVersionsOnce initialize version once
func InitVersionsOnce(state util.MerklePatriciaTrieI) error {
	var initFuncs []initVersionFunc
	for _, f := range isStateReadyCheckFuncs {
		initFunc, err := f(state)
		if err != nil {
			return err
		}

		initFuncs = append(initFuncs, initFunc)
	}

	initOnce.Do(func() {
		for _, f := range initFuncs {
			f()
		}
	})

	return nil
}

type isStateReadyCheckFunc func(util.MerklePatriciaTrieI) (initVersionFunc, error)

type initVersionFunc func()

func newVersion(f isStateReadyCheckFunc) *Version {
	v := &Version{lock: &sync.RWMutex{}}
	register(v, f)
	return v
}

func register(v *Version, f isStateReadyCheckFunc) {
	versions = append(versions, v)
	isStateReadyCheckFuncs = append(isStateReadyCheckFuncs, f)
}

type Version struct {
	v    semver.Version
	lock *sync.RWMutex
}

// IsEmpty checks if the version is empty
func (scv *Version) IsEmpty() bool {
	return !scv.Get().Equals(emptyVersion)
}

func (scv *Version) Set(v semver.Version) {
	scv.lock.Lock()
	scv.v = v
	scv.lock.Unlock()
}

func (scv *Version) Get() semver.Version {
	scv.lock.RLock()
	v := scv.v
	scv.lock.RUnlock()
	return v
}

func (scv *Version) String() string {
	scv.lock.RLock()
	s := scv.v.String()
	scv.lock.RUnlock()
	return s
}
