package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/smartcontract"
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

	latestSCVersion, err := semver.Make(smartcontract.LatestSupportedSCVersion)
	if err != nil {
		logging.Logger.Panic(fmt.Sprintf("start_versions_worker, invalid latest supported sc version: %v", err))
		return
	}

	currentSCVersion := smartcontract.GetSCVersion()

	if latestSCVersion.LE(currentSCVersion) {
		logging.Logger.Debug("start_versions_worker exit, no new sc version detected",
			zap.String("current sc version", currentSCVersion.String()),
			zap.String("latest sc version", latestSCVersion.String()))
		return
	}

	// sign the version
	self := node.Self

	versions := &VersionsEntity{
		SCVersion: latestSCVersion.String(),
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
	SCVersion string `json:"sc_version"`
	Sign      string `json:"sign"`
}

func (v *VersionsEntity) GetEntityMetadata() datastore.EntityMetadata {
	return versionsEntityMetaData
}

func (v *VersionsEntity) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", v.SCVersion))
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
	return encryption.Hash(v.SCVersion)
}
