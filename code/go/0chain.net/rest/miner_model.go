package rest

import (
	"errors"
	"net/url"

	"0chain.net/chaincore/node"
	"0chain.net/smartcontract/dbs/event"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"

	"0chain.net/smartcontract/minersc"
)

func decodeFromValues(mn *minersc.MinerNode, params url.Values) error {
	mn.N2NHost = params.Get("n2n_host")
	mn.ID = params.Get("id")

	if mn.N2NHost == "" || mn.ID == "" {
		return errors.New("URL or ID is not specified")
	}
	return nil
}

func doesMinerExist(pkey datastore.Key, balances cstate.ReadOnlyStateContextI) bool {
	mn := minersc.NewMinerNode()
	err := balances.GetTrieNode(pkey, mn)
	if err != nil {
		if err != util.ErrValueNotPresent {
			logging.Logger.Error("GetTrieNode from state context", zap.Error(err),
				zap.String("key", pkey))
		}
		return false
	}

	return true
}

func minerTableToMinerNode(edbMiner event.Miner) minersc.MinerNode {
	var status = node.NodeStatusInactive
	if edbMiner.Active {
		status = node.NodeStatusActive
	}
	msn := minersc.SimpleNode{
		ID:                edbMiner.MinerID,
		N2NHost:           edbMiner.N2NHost,
		Host:              edbMiner.Host,
		Port:              edbMiner.Port,
		Path:              edbMiner.Path,
		PublicKey:         edbMiner.PublicKey,
		ShortName:         edbMiner.ShortName,
		BuildTag:          edbMiner.BuildTag,
		TotalStaked:       int64(edbMiner.TotalStaked),
		Delete:            edbMiner.Delete,
		DelegateWallet:    edbMiner.DelegateWallet,
		ServiceCharge:     edbMiner.ServiceCharge,
		NumberOfDelegates: edbMiner.NumberOfDelegates,
		MinStake:          edbMiner.MinStake,
		MaxStake:          edbMiner.MaxStake,
		Stat: minersc.Stat{
			GeneratorRewards: edbMiner.Rewards,
			GeneratorFees:    edbMiner.Fees,
		},
		Geolocation: minersc.SimpleNodeGeolocation{
			Latitude:  edbMiner.Latitude,
			Longitude: edbMiner.Longitude,
		},
		NodeType:        minersc.NodeTypeMiner,
		LastHealthCheck: edbMiner.LastHealthCheck,
		Status:          status,
	}

	return minersc.MinerNode{
		SimpleNode: &msn,
	}
}

func sharderTableToSharderNode(edbSharder event.Sharder) minersc.MinerNode {
	var status = node.NodeStatusInactive
	if edbSharder.Active {
		status = node.NodeStatusActive
	}
	msn := minersc.SimpleNode{
		ID:                edbSharder.SharderID,
		N2NHost:           edbSharder.N2NHost,
		Host:              edbSharder.Host,
		Port:              edbSharder.Port,
		Path:              edbSharder.Path,
		PublicKey:         edbSharder.PublicKey,
		ShortName:         edbSharder.ShortName,
		BuildTag:          edbSharder.BuildTag,
		TotalStaked:       int64(edbSharder.TotalStaked),
		Delete:            edbSharder.Delete,
		DelegateWallet:    edbSharder.DelegateWallet,
		ServiceCharge:     edbSharder.ServiceCharge,
		NumberOfDelegates: edbSharder.NumberOfDelegates,
		MinStake:          edbSharder.MinStake,
		MaxStake:          edbSharder.MaxStake,
		Stat: minersc.Stat{
			GeneratorRewards: edbSharder.Rewards,
			GeneratorFees:    edbSharder.Fees,
		},
		LastHealthCheck: edbSharder.LastHealthCheck,
		Geolocation: minersc.SimpleNodeGeolocation{
			Latitude:  edbSharder.Latitude,
			Longitude: edbSharder.Longitude,
		},
		NodeType: minersc.NodeTypeSharder,
		Status:   status,
	}

	return minersc.MinerNode{
		SimpleNode: &msn,
	}
}

type DelegatePoolStat struct {
	ID           string        `json:"id"`            // pool ID
	Balance      state.Balance `json:"balance"`       //
	InterestPaid state.Balance `json:"interest_paid"` //
	RewardPaid   state.Balance `json:"reward_paid"`   //
	Status       string        `json:"status"`        //
	High         state.Balance `json:"high"`          // }
	Low          state.Balance `json:"low"`           // }
}

func newDelegatePoolStat(dp *sci.DelegatePool) (dps *DelegatePoolStat) {
	dps = new(DelegatePoolStat)
	dps.ID = dp.ID
	dps.Balance = dp.Balance
	dps.InterestPaid = dp.InterestPaid
	dps.RewardPaid = dp.RewardPaid
	dps.Status = dp.Status
	dps.High = dp.High
	dps.Low = dp.Low
	return
}

// A userPools represents response for user pools requests.
// swagger:model userPools
type userPools struct {
	Pools map[string]map[string][]*DelegatePoolStat `json:"pools"`
}

func newUserPools() (ups *userPools) {
	ups = new(userPools)
	ups.Pools = make(map[string]map[string][]*DelegatePoolStat)
	return
}

// swagger:model events
type eventList struct {
	Events []event.Event `json:"events"`
}
