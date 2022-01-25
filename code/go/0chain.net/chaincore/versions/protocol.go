package versions

import (
	"sync"

	"0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/blang/semver/v4"
	"go.uber.org/zap"
)

var protoVersion = newProtoVersion()

func newProtoVersion() *Version {
	v := &Version{
		lock: &sync.RWMutex{},
		//updateVersionFunc: updateProtoVersionWithState,
	}

	//v.isStateReadyFunc = checkProtoVersionOrInit(v)

	register(v)
	return v
}

// checkProtoVersionOrInit checks MPT state to see if protocol version could be found,
// return a function to set protocol version with the value from state
//func checkProtoVersionOrInit(pv *Version) isStateReadyCheckFunc {
//	return func(state util.MerklePatriciaTrieI) (initVersionFunc, error) {
//		v, err := GetProtoVersionFromState(state)
//		if err != nil {
//			return nil, common.NewErrorf("init_proto_version_failed",
//				"could not get protocol version from MPT: %v", err)
//		}
//
//		return func() {
//			pv.Set(*v)
//			logging.Logger.Debug("init protocol version", zap.String("version", v.String()))
//		}, nil
//	}
//}

func CheckProtoVersionAndSetDelay(getVersion GetVersionFunc) CheckVersionAndSetDelayFunc {
	return func(state util.MerklePatriciaTrieI) (SetVersionFunc, error) {
		v, err := getVersion(state)
		if err != nil {
			return nil, err
		}

		return func() {
			protoVersion.Set(*v)
		}, nil
	}
}

func UpdateProtoVersion(getVersion GetVersionFunc) UpdateVersionFunc {
	return func(state util.MerklePatriciaTrieI) error {
		v, err := getVersion(state)
		if err != nil {
			return err
		}

		if v.GT(protoVersion.Get()) {
			protoVersion.Set(*v)
			logging.Logger.Debug("updated protocol version",
				zap.String("previous version", protoVersion.String()),
				zap.String("new version", v.String()))
		}
		return nil
	}
}

//func updateProtoVersionWithState(cv *Version, state util.MerklePatriciaTrieI) error {
//	v, err := GetProtoVersionFromState(state)
//	if err != nil {
//		return err
//	}
//
//	if v.GT(cv.Get()) {
//		cv.Set(*v)
//		logging.Logger.Debug("updated protocol version",
//			zap.String("previous version", cv.String()),
//			zap.String("new version", v.String()))
//	}
//	return nil
//}

// GetProtoVersion returns protocol version
func GetProtoVersion() semver.Version {
	return protoVersion.Get()
}

func SetProtoVersion(v *semver.Version) {
	protoVersion.Set(*v)
}
