package chain

import (
	"context"
	"strconv"

	"0chain.net/chaincore/threshold/bls"
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

func (c *Chain) GetDKGByStartingRound(round int64) *bls.DKG {
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

func StoreDKGSummary(ctx context.Context, dkgSummary *bls.DKGSummary) error {
	dkgs := datastore.GetEntity("dkgsummary").(*bls.DKGSummary)
	dkgSummaryMetadata := dkgs.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, dkgSummaryMetadata)
	defer ememorystore.Close(dctx)
	return dkgSummary.Write(dctx)
}

type deleteAddNodes struct {
	Deleted []string
	Added   []string
}

// func NewDKGWithMagicBlock(mb *block.MagicBlock, summary *bls.DKGSummary) (*bls.DKG, *deleteAddNodes, error) {
// 	selfNodeKey := node.Self.Underlying().GetKey()

// 	// if summary.SecretShares == nil {
// 	// 	return nil, nil, common.NewError("failed to set dkg from store", "no saved shares for dkg")
// 	// }

// 	// bls.SetDKG(mb.T, mb.N, summary.SecretShares, summary)

// 	var newDKG = bls.MakeDKG(mb.T, mb.N, selfNodeKey)
// 	newDKG.MagicBlockNumber = mb.MagicBlockNumber
// 	newDKG.StartingRound = mb.StartingRound

// 	if mb.Miners == nil {
// 		return nil, nil, common.NewError("failed to set dkg from store", "miners pool is not initialized in magic block")
// 	}

// 	minerNodes := mb.Miners.CopyNodesMap()
// 	var (
// 		daNodes deleteAddNodes
// 		ids     = make([]bls.PartyID, 0, len(minerNodes))
// 	)

// 	for mid := range minerNodes {
// 		pid := bls.ComputeIDdkg(mid)
// 		ids = append(ids, pid)
// 	}

// 	// generate Sij for miners
// 	for _, id := range ids {
// 		_, err := newDKG.ComputeDKGKeyShare(id)
// 		if err != nil {
// 			return nil, nil, fmt.Errorf("failed to compute sij: %v", err)
// 		}
// 	}

// 	// for mid := range minerNodes {
// 	// 	pid := bls.ComputeIDdkg(mid)
// 	// 	k := pid.GetHexString()
// 	// 	ids = append(ids, pid)
// 	// 	logging.Logger.Debug("new dkg from magic block", zap.String("key", k), zap.Any("summary shares", summary.SecretShares))
// 	// 	if savedShare, ok := summary.SecretShares[k]; ok {
// 	// 		if err := newDKG.AddSecretShare(pid, savedShare, false); err != nil {
// 	// 			return nil, nil, err
// 	// 		}
// 	// 		logging.Logger.Debug("new dkg from magic block", zap.String("key", k), zap.String("share", savedShare))
// 	// 	} else if v, ok := mb.GetShareOrSigns().Get(k); ok {
// 	// 		daNodes.Added = append(daNodes.Added, k)
// 	// 		if share, ok := v.ShareOrSigns[node.Self.Underlying().GetKey()]; ok && share.Share != "" {
// 	// 			if err := newDKG.AddSecretShare(pid, share.Share, false); err != nil {
// 	// 				return nil, nil, err
// 	// 			}
// 	// 		}
// 	// 	}
// 	// }

// 	// for k := range summary.SecretShares {
// 	// 	if _, ok := minerNodes[k]; !ok {
// 	// 		daNodes.Deleted = append(daNodes.Deleted, k)
// 	// 	}
// 	// }

// 	if !newDKG.HasAllSecretShares() {
// 		logging.Logger.Error("not enough secret shares for dkg",
// 			zap.Int("new DKG T", newDKG.T),
// 			zap.Int("total secret shares", len(newDKG.GetSecretKeyShares())))
// 		return nil, nil, common.NewError("failed to set dkg from store",
// 			"not enough secret shares for dkg")
// 	}

// 	newDKG.AggregateSecretKeyShares()
// 	newDKG.Pi = newDKG.Si.GetPublicKey()
// 	logging.Logger.Debug("dkg PI", zap.String("key", newDKG.Pi.GetHexString()))
// 	mpks, err := mb.Mpks.GetMpkMap()
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	if err := newDKG.AggregatePublicKeyShares(mpks); err != nil {
// 		return nil, nil, err
// 	}

// 	return newDKG, &daNodes, nil
// }

// func (c *Chain) SetDKGFromPreviousSummary(ctx context.Context, mb *block.MagicBlock) error {
// 	summary, err := LoadDKGSummary(ctx, mb.MagicBlockNumber-1)
// 	if err != nil {
// 		return err
// 	}

// 	newDKG, deleteAddNodes, err := NewDKGWithMagicBlock(mb, summary)
// 	if err != nil {
// 		return err
// 	}

// 	if len(deleteAddNodes.Deleted) > 0 {
// 		for _, k := range deleteAddNodes.Deleted {
// 			delete(summary.SecretShares, k)
// 		}
// 	}

// 	if err := StoreDKGSummary(ctx, summary); err != nil {
// 		return fmt.Errorf("failed to store dkg summary: %v", err)
// 	}

// 	if err = c.SetDKG(newDKG); err != nil {
// 		logging.Logger.Error("failed to set dkg", zap.Error(err))
// 		return err // error
// 	}
// 	return nil
// }

// ComputeBlsID Handy API to get the ID used in the library
func ComputeBlsID(key string) string {
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetHexString()
}
