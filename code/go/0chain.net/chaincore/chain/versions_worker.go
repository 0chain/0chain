package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/chaincore/smartcontract"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
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

	go func() {
		defer close(doneC)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				// broadcast sc version report message
				c.SendVersions(context.Background(), &VersionsEntity{
					SCVersion: smartcontract.LatestSupportedSCVersion,
				})
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
	//
	Share string `json:"share"`
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
