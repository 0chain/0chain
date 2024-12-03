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
