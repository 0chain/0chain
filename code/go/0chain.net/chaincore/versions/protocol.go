package versions

import (
	"sync"

	bcstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"
	"github.com/blang/semver/v4"
	"go.uber.org/zap"
)

const LatestSupportedProtoVersion = "2.0.0"

// TODO: comment out this to register protocol version
var protoVersion = newProtoVersion()

func newProtoVersion() *Version {
	v := &Version{lock: &sync.RWMutex{}}
	register(v, checkProtoVersionOrInit(v))
	return v
}

// checkProtoVersionOrInit checks MPT state to see if protocol version could be found,
// return a function to set protocol version with the value from state
func checkProtoVersionOrInit(pv *Version) isStateReadyCheckFunc {
	return func(state util.MerklePatriciaTrieI) (initVersionFunc, error) {
		v, err := GetProtoVersionFromState(state)
		if err != nil {
			return nil, common.NewErrorf("init_proto_version_failed",
				"could not get protocol version from MPT: %v", err)
		}

		return func() {
			pv.Set(*v)
			logging.Logger.Debug("init protocol version", zap.String("version", v.String()))
		}, nil
	}
}

// GetProtoVersion returns protocol version
func GetProtoVersion() semver.Version {
	return protoVersion.Get()
}

func SetProtoVersion(v *semver.Version) {
	protoVersion.Set(*v)
}

func GetProtoVersionFromState(state util.MerklePatriciaTrieI) (*semver.Version, error) {
	vn, err := bcstate.GetTrieNode(state, minersc.ProtoVersionKey)
	if err != nil {
		return nil, err
	}

	var vnode minersc.VersionNode
	if err := vnode.Decode(vn.Encode()); err != nil {
		return nil, err
	}

	return semver.New(vnode.String())
}
