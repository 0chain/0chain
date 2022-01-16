package chain

import (
	"context"
	"encoding/json"
	"time"

	"0chain.net/chaincore/node"
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

	newSCVersion := smartcontract.GetNewVersion(versions.GetSCVersion())

	// sign the version
	self := node.Self

	versions := NewVersionsEntity()
	if newSCVersion != nil {
		versions.Add("sc_version", *newSCVersion)
	}

	sign, err := self.Sign(versions.Hash())
	if err != nil {
		logging.Logger.Error("start_versions_worker, failed to sign versions",
			zap.Error(err))
		return
	}

	versions.Sign = sign

	go func() {
		defer close(doneC)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				// broadcast sc version report message
				logging.Logger.Debug("report versions", zap.Any("versions", versions.Versions))
				c.SendVersions(context.Background(), versions)
			}
		}
	}()

	<-doneC
	return
}

var versionsEntityMetaData *datastore.EntityMetadataImpl

type VersionsEntity struct {
	datastore.NOIDField
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
	return &VersionsEntity{}
}

func SetupVersionsEntity() {
	versionsEntityMetaData = datastore.MetadataProvider()
	versionsEntityMetaData.Name = "versions"
	versionsEntityMetaData.Provider = VersionsEntityProvider

	datastore.RegisterEntityMetadata("versions", versionsEntityMetaData)
}

func (v *VersionsEntity) Hash() string {
	if v.Versions == nil {
		return ""
	}

	d, err := json.Marshal(v.Versions)
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
