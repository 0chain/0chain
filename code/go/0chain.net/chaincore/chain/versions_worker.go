package chain

import (
	"context"
	"encoding/json"
	"time"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/protocol"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/versions"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"github.com/blang/semver/v4"
	"go.uber.org/zap"
)

func init() {
	SetupVersionsEntity()
}

var (
	VersionEntitySCKey             = "sc_version"
	VersionEntityFinalizedSCKey    = "finalized_sc_version"
	VersionEntityProtoKey          = "proto_version"
	VersionEntityFinalizedProtoKey = "finalized_proto_version"
)

func (c *Chain) SendVersions(ctx context.Context, v *VersionsEntity) {
	mb := c.GetMagicBlock(c.GetCurrentRound())
	if mb.Miners == nil {
		logging.Logger.Error("could not send versions, no miners found in magic block")
		return
	}

	sender := VersionsSender(v)

	mb.Miners.SendAll(ctx, sender)
	mb.Sharders.SendAll(ctx, sender)
}

func StartVersionsWorker(ctx context.Context, c Chainer) {
	doneC := make(chan struct{})
	tk := time.NewTicker(time.Minute)

	ve := NewVersionsEntity()

	newSCVersion := getNewVersion(versions.GetSCVersion(), smartcontract.LatestSupportSCVersion)
	if newSCVersion != nil {
		ve.Add(VersionEntitySCKey, *newSCVersion)
	}

	newProtoVersion := getNewVersion(versions.GetProtoVersion(), protocol.LatestSupportProtoVersion)
	if newProtoVersion != nil {
		ve.Add(VersionEntityProtoKey, *newProtoVersion)
	}

	sign, err := node.Self.Sign(ve.Hash())
	if err != nil {
		logging.Logger.Error("start_versions_worker, failed to sign versions",
			zap.Error(err))
		return
	}

	ve.Sign = sign

	go func() {
		defer close(doneC)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				//// get latest finalized sc_version in MPT
				//finalizedSCVersion, err := getSCVersionFromState(c.GetLatestFinalizedState())
				//if err == nil {
				//	ve.Add(VersionEntityFinalizedProtoKey, *finalizedSCVersion)
				//} else {
				//	logging.Logger.Error("report versions, could not found sc version in state", zap.Error(err))
				//}
				//
				//finalizedProtoVersion, err := getProtoVersionFromState(c.GetLatestFinalizedState())
				//if err == nil {
				//	ve.Add(VersionEntityFinalizedProtoKey, *finalizedProtoVersion)
				//} else {
				//	logging.Logger.Error("report versions, could not found sc version in state", zap.Error(err))
				//}

				// broadcast sc version report message
				logging.Logger.Debug("report versions", zap.Any("versions", ve.Versions))
				c.SendVersions(context.Background(), ve)
			}
		}
	}()

	<-doneC
	return
}

var versionsEntityMetaData *datastore.EntityMetadataImpl

type VersionsEntity struct {
	datastore.NOIDField
	datastore.VersionField
	datastore.NoProtocolChange
	Versions map[string]string `json:"versions"`
	Sign     string            `json:"sign"`
}

func (v *VersionsEntity) GetEntityMetadata() datastore.EntityMetadata {
	return versionsEntityMetaData
}

func (v *VersionsEntity) GetKey() datastore.Key {
	return datastore.ToKey(v.Sign)
}

func VersionsEntityProvider() datastore.Entity {
	v := &VersionsEntity{}
	v.Version = protocol.LatestSupportProtoVersion.String()
	return v
}

func SetupVersionsEntity() {
	versionsEntityMetaData = datastore.MetadataProvider()
	versionsEntityMetaData.Name = "versions"
	versionsEntityMetaData.Provider = VersionsEntityProvider

	datastore.RegisterEntityMetadata("versions", versionsEntityMetaData)
}

// Hash should not including the Sign field
func (v *VersionsEntity) Hash() string {
	var ve = struct {
		datastore.VersionField
		Versions map[string]string `json:"versions"`
	}{
		Versions: v.Versions,
	}

	ve.Version = v.Version

	d, err := json.Marshal(ve)
	if err != nil {
		panic(err)
	}

	return encryption.Hash(string(d))
}

func NewVersionsEntity() *VersionsEntity {
	return &VersionsEntity{
		Versions: make(map[string]string),
	}
}

func (v *VersionsEntity) Add(name string, version semver.Version) {
	v.Versions[name] = version.String()
}

func (v *VersionsEntity) Get(name string) (*semver.Version, error) {
	s, ok := v.Versions[name]
	if !ok {
		logging.Logger.Debug("version not exist", zap.String("name", name),
			zap.Any("versions", v.Versions))
		return nil, nil
	}

	return semver.New(s)
}

// getNewVersion returns the new version if the latest support version is
// greater than the current running version, otherwise return nil to indicate
// no new version detected.
func getNewVersion(currentVersion semver.Version, latestSupportVersion semver.Version) *semver.Version {
	if latestSupportVersion.LE(currentVersion) {
		logging.Logger.Debug("start_versions_worker exit, no new sc version detected",
			zap.String("current version", currentVersion.String()),
			zap.String("latest support version", latestSupportVersion.String()))
		return nil
	}

	return &latestSupportVersion
}
