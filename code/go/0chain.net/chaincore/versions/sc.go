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

// define and register the smart contract version
var scVersion = newSCVersion()

func newSCVersion() *Version {
	v := &Version{lock: &sync.RWMutex{}}
	register(v, checkSCVersionOrInit(v))
	return v
}

// checkSCVersionOrInit checks MPT state to see if sc version could be found,
// return a function to set sc version with the value from state
func checkSCVersionOrInit(scv *Version) isStateReadyCheckFunc {
	return func(state util.MerklePatriciaTrieI) (initVersionFunc, error) {
		v, err := GetSCVersionFromState(state)
		if err != nil {
			return nil, common.NewErrorf("init_sc_version_failed", "could not get sc version from MPT: %v", err)
		}

		return func() {
			scv.Set(*v)
			logging.Logger.Debug("init sc version", zap.String("version", v.String()))
		}, nil
	}
}

// GetSCVersion returns smart contract version
func GetSCVersion() semver.Version {
	return scVersion.Get()
}

// SetSCVersion sets smart contract version
func SetSCVersion(v *semver.Version) {
	scVersion.Set(*v)
}

func GetSCVersionFromState(state util.MerklePatriciaTrieI) (*semver.Version, error) {
	vn, err := bcstate.GetTrieNode(state, minersc.SCVersionKey)
	if err != nil {
		return nil, err
	}

	var vnode minersc.VersionNode
	if err := vnode.Decode(vn.Encode()); err != nil {
		return nil, err
	}

	return semver.New(vnode.String())
}
