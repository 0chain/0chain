package storagesc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/currency"
)

var blobbersInfoListName = encryption.Hash(ADDRESS + "blobbers_info_list")

//go:generate msgp -io=false -tests=false -v
type BlobberOfferStake struct {
	TotalOffers currency.Coin `msg:"o"`
	TotalStake  currency.Coin `msg:"s"`
	Allocated   int64         `msg:"a"`
}

type BlobberOfferStakeList []*BlobberOfferStake

func getBlobbersInfoList(balance cstate.StateContextI) (BlobberOfferStakeList, error) {
	var bil BlobberOfferStakeList
	if err := balance.GetTrieNode(blobbersInfoListName, &bil); err != nil {
		return nil, fmt.Errorf("could not get blobbers info list: %v", err)
	}
	return bil, nil
}

func (bi *BlobberOfferStakeList) Add(b *BlobberOfferStake) int32 {
	*bi = append(*bi, b)
	return int32(len(*bi)) - 1
}

func (bi *BlobberOfferStakeList) Save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(blobbersInfoListName, bi)
	if err != nil {
		return fmt.Errorf("could not save blobbers info list: %v", err)
	}
	return nil
}

func (bi *BlobberOfferStake) addOffer(amount currency.Coin) error {
	newTotalOffers, err := currency.AddCoin(bi.TotalOffers, amount)
	if err != nil {
		return err
	}
	bi.TotalOffers = newTotalOffers
	return nil
}

// add offer of an allocation related to blobber owns this stake pool
func (bi *BlobberOfferStake) reduceOffer(amount currency.Coin) error {
	newTotalOffers, err := currency.MinusCoin(bi.TotalOffers, amount)
	if err != nil {
		return err
	}
	bi.TotalOffers = newTotalOffers
	return nil
}
