package chain

import (
	"context"
	"strconv"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

// GetDKG returns DKG by round number.
func (c *Chain) GetDKG(round int64) *bls.DKG {

	round = mbRoundOffset(round)

	c.roundDkgMu.RLock()
	defer c.roundDkgMu.RUnlock()
	entity := c.roundDkg.Get(round)
	if entity == nil {
		return nil
	}
	return entity.(*bls.DKG)
}

// SetDKG sets DKG for the start round
func (c *Chain) SetDKG(dkg *bls.DKG) error {
	c.roundDkgMu.Lock()
	defer c.roundDkgMu.Unlock()
	return c.roundDkg.Put(dkg, dkg.StartingRound)
}

// LoadDKGSummary loads DKG summary by stored DKG (that stores DKG summary).
func LoadDKGSummary(ctx context.Context, id int64) (dkgs *bls.DKGSummary, err error) {
	dkgs = datastore.GetEntity("dkgsummary").(*bls.DKGSummary)
	dkgs.ID = strconv.FormatInt(id, 10)
	var (
		dkgSummaryMetadata = dkgs.GetEntityMetadata()
		dctx               = ememorystore.WithEntityConnection(ctx,
			dkgSummaryMetadata)
	)
	defer ememorystore.Close(dctx)
	err = dkgs.Read(dctx, dkgs.GetKey())
	return
}

func NewDKGWithMagicBlock(mb *block.MagicBlock, summary *bls.DKGSummary) (*bls.DKG, error) {
	selfNodeKey := node.Self.Underlying().GetKey()

	if summary.SecretShares == nil {
		return nil, common.NewError("failed to set dkg from store", "no saved shares for dkg")
	}

	var newDKG = bls.MakeDKG(mb.T, mb.N, selfNodeKey)
	newDKG.MagicBlockNumber = mb.MagicBlockNumber
	newDKG.StartingRound = mb.StartingRound

	if mb.Miners == nil {
		return nil, common.NewError("failed to set dkg from store", "miners pool is not initialized in magic block")
	}

	for k := range mb.Miners.CopyNodesMap() {
		if savedShare, ok := summary.SecretShares[ComputeBlsID(k)]; ok {
			if err := newDKG.AddSecretShare(bls.ComputeIDdkg(k), savedShare, false); err != nil {
				return nil, err
			}
		}
		//  else if v, ok := mb.GetShareOrSigns().Get(k); ok {
		// 	if share, ok := v.ShareOrSigns[node.Self.Underlying().GetKey()]; ok && share.Share != "" {
		// 		if err := newDKG.AddSecretShare(bls.ComputeIDdkg(k), share.Share, false); err != nil {
		// 			return nil, err
		// 		}
		// 	}
		// }
	}

	if !newDKG.HasAllSecretShares() {
		return nil, common.NewError("failed to set dkg from store",
			"not enough secret shares for dkg")
	}

	newDKG.AggregateSecretKeyShares()
	newDKG.Pi = newDKG.Si.GetPublicKey()
	mpks, err := mb.Mpks.GetMpkMap()
	if err != nil {
		return nil, err
	}

	if err := newDKG.AggregatePublicKeyShares(mpks); err != nil {
		return nil, err
	}

	return newDKG, nil
}

// func (c *Chain) SetDKGSFromStore(ctx context.Context, mb *block.MagicBlock) (err error) {
// 	summary, err := LoadDKGSummary(ctx, mb.MagicBlockNumber)
// 	if err != nil {
// 		return err
// 	}
// 	newDKG, err := NewDKGWithMagicBlock(mb, summary)
// 	if err = c.SetDKG(newDKG, mb.StartingRound); err != nil {
// 		logging.Logger.Error("failed to set dkg", zap.Error(err))
// 		return // error
// 	}

// 	return // ok, set
// }

// ComputeBlsID Handy API to get the ID used in the library
func ComputeBlsID(key string) string {
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetHexString()
}
