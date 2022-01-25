package versions

import (
	"sync"

	"0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/blang/semver/v4"
	"go.uber.org/zap"
)

// define and register the smart contract version
var scVersion = newSCVersion()

func newSCVersion() *Version {
	v := &Version{
		lock: &sync.RWMutex{},
	}

	//v.isStateReadyFunc = checkSCVersionOrInit(v)
	//v.updateVersionFunc = updateSCVersionWithState

	register(v)
	return v
}

func CheckSCVersionAndSetDelay(getVersion GetVersionFunc) CheckVersionAndSetDelayFunc {
	return func(state util.MerklePatriciaTrieI) (SetVersionFunc, error) {
		v, err := getVersion(state)
		if err != nil {
			return nil, err
		}

		return func() {
			scVersion.Set(*v)
		}, nil
	}
}

func UpdateSCVersion(getVersion GetVersionFunc) UpdateVersionFunc {
	return func(state util.MerklePatriciaTrieI) error {
		v, err := getVersion(state)
		if err != nil {
			return err
		}

		if v.GT(scVersion.Get()) {
			scVersion.Set(*v)
			logging.Logger.Debug("updated sc version",
				zap.String("previous version", scVersion.String()),
				zap.String("new version", v.String()))
		}
		return nil
	}
}

// checkSCVersionOrInit checks MPT state to see if sc version could be found,
// return a function to set sc version with the value from state
//func checkSCVersionOrInit(scv *Version) isStateReadyCheckFunc {
//	return func(state util.MerklePatriciaTrieI) (initVersionFunc, error) {
//		v, err := GetSCVersionFromState(state)
//		if err != nil {
//			return nil, common.NewErrorf("init_sc_version_failed", "could not get sc version from MPT: %v", err)
//		}
//
//		return func() {
//			scv.Set(*v)
//			logging.Logger.Debug("init sc version", zap.String("version", v.String()))
//		}, nil
//	}
//}
//
//func updateSCVersionWithState(cv *Version, state util.MerklePatriciaTrieI) error {
//	v, err := GetSCVersionFromState(state)
//	if err != nil {
//		return err
//	}
//
//	if v.GT(cv.Get()) {
//		cv.Set(*v)
//		logging.Logger.Debug("updated sc version",
//			zap.String("previous version", cv.String()),
//			zap.String("new version", v.String()))
//	}
//	return nil
//}

// GetSCVersion returns smart contract version
func GetSCVersion() semver.Version {
	return scVersion.Get()
}

// SetSCVersion sets smart contract version
func SetSCVersion(v *semver.Version) {
	scVersion.Set(*v)
}
