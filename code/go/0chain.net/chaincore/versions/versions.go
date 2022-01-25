package versions

import (
	"sync"

	"0chain.net/core/util"
	"github.com/blang/semver/v4"
)

var (
	emptyVersion semver.Version

	// registered versions
	versions []Versioner

	initOnce sync.Once
)

//type (
//isStateReadyCheckFunc      func(util.MerklePatriciaTrieI) (initVersionFunc, error)
//updateVersionWithStateFunc func(*Version, util.MerklePatriciaTrieI) error
//initVersionFunc func()
//)

type (
	UpdateVersionFunc           func(util.MerklePatriciaTrieI) error
	GetVersionFunc              func(util.MerklePatriciaTrieI) (*semver.Version, error)
	SetVersionFunc              func()
	CheckVersionAndSetDelayFunc func(util.MerklePatriciaTrieI) (SetVersionFunc, error)
)

type Versioner interface {
	IsEmpty() bool
	Set(version semver.Version)
	Get() semver.Version
	String() string

	//isStateReady(util.MerklePatriciaTrieI) (initVersionFunc, error)
	//updateVersionWithState(util.MerklePatriciaTrieI) error
}

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
func InitVersionsOnce(state util.MerklePatriciaTrieI, fs ...CheckVersionAndSetDelayFunc) error {
	var setFuncs []func()
	for _, f := range fs {
		setFunc, err := f(state)
		if err != nil {
			return err
		}

		setFuncs = append(setFuncs, setFunc)
	}

	initOnce.Do(func() {
		for _, f := range setFuncs {
			f()
		}
	})

	return nil
}

// UpdateVersionsWithState updates versions
func UpdateVersionsWithState(state util.MerklePatriciaTrieI, fs ...UpdateVersionFunc) error {
	for _, f := range fs {
		if err := f(state); err != nil {
			return err
		}
	}

	return nil
}

func register(v *Version) {
	versions = append(versions, v)
}

type Version struct {
	v    semver.Version
	lock *sync.RWMutex

	//isStateReadyFunc  isStateReadyCheckFunc
	//updateVersionFunc updateVersionWithStateFunc
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

//func (scv *Version) isStateReady(state util.MerklePatriciaTrieI) (initVersionFunc, error) {
//	return scv.isStateReadyFunc(state)
//}
//
//func (scv *Version) updateVersionWithState(state util.MerklePatriciaTrieI) error {
//	return scv.updateVersionFunc(scv, state)
//}
