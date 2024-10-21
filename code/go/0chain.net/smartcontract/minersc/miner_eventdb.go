package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
)

type SimpleNodeResponse struct {
	ID                            string           `json:"id" validate:"hexadecimal,len=64"`
	N2NHost                       string           `json:"n2n_host"`
	Host                          string           `json:"host"`
	Port                          int              `json:"port"`
	Path                          string           `json:"path"`
	PublicKey                     string           `json:"public_key"`
	ShortName                     string           `json:"short_name"`
	BuildTag                      string           `json:"build_tag"`
	TotalStaked                   currency.Coin    `json:"total_stake"`
	Delete                        bool             `json:"delete"`
	NodeType                      NodeType         `json:"node_type,omitempty"`
	LastHealthCheck               common.Timestamp `json:"last_health_check"`
	Status                        int              `json:"-" msg:"-"`
	LastSettingUpdateRound        int64            `json:"last_setting_update_round"`
	RoundServiceChargeLastUpdated int64            `json:"round_service_charge_last_updated"`
	IsKilled                      bool             `json:"is_killed"`
}

type DelegatePoolResponse struct {
	stakepool.DelegatePool
	RoundPoolLastUpdated int64 `json:"round_pool_last_updated"`
}

type StakePoolResponse struct {
	Pools    map[string]*DelegatePoolResponse `json:"pools"`
	Reward   currency.Coin                    `json:"rewards"`
	Settings stakepool.Settings               `json:"settings"`
	Minter   cstate.ApprovedMinter            `json:"minter"`
}

type NodeResponse struct {
	*SimpleNodeResponse `json:"simple_miner"`
	*StakePoolResponse  `json:"stake_pool"`
}

func minerTableToMinerNode(edbMiner event.Miner, delegates []event.DelegatePool) NodeResponse {
	var status = node.NodeStatusInactive
	if edbMiner.Active {
		status = node.NodeStatusActive
	}
	msn := SimpleNodeResponse{
		ID:                            edbMiner.ID,
		N2NHost:                       edbMiner.N2NHost,
		Host:                          edbMiner.Host,
		Port:                          edbMiner.Port,
		Path:                          edbMiner.Path,
		PublicKey:                     edbMiner.PublicKey,
		ShortName:                     edbMiner.ShortName,
		BuildTag:                      edbMiner.BuildTag,
		TotalStaked:                   edbMiner.Provider.TotalStake,
		Delete:                        edbMiner.Delete,
		NodeType:                      NodeTypeMiner,
		LastHealthCheck:               edbMiner.LastHealthCheck,
		Status:                        status,
		RoundServiceChargeLastUpdated: edbMiner.Rewards.RoundServiceChargeLastUpdated,
		IsKilled:                      edbMiner.IsKilled,
	}

	mn := NodeResponse{
		SimpleNodeResponse: &msn,
		StakePoolResponse: &StakePoolResponse{
			Reward: edbMiner.Rewards.Rewards,
			Settings: stakepool.Settings{
				DelegateWallet:     edbMiner.DelegateWallet,
				ServiceChargeRatio: edbMiner.ServiceCharge,
				MaxNumDelegates:    edbMiner.Provider.NumDelegates,
			},
		},
	}
	if len(delegates) == 0 {
		return mn
	}
	mn.StakePoolResponse.Pools = make(map[string]*DelegatePoolResponse)
	for _, delegate := range delegates {
		mn.StakePoolResponse.Pools[delegate.PoolID] = &DelegatePoolResponse{
			DelegatePool: stakepool.DelegatePool{
				Balance:      delegate.Balance,
				Reward:       delegate.Reward,
				Status:       delegate.Status,
				RoundCreated: delegate.RoundCreated,
				DelegateID:   delegate.DelegateID,
				StakedAt:     delegate.StakedAt,
			},
			RoundPoolLastUpdated: delegate.RoundPoolLastUpdated,
		}
	}
	return mn
}

func minerNodeToMinerTable(mn *MinerNode, round int64) event.Miner {
	return event.Miner{
		N2NHost:   mn.N2NHost,
		Host:      mn.Host,
		Port:      mn.Port,
		Path:      mn.Path,
		PublicKey: mn.PublicKey,
		ShortName: mn.ShortName,
		BuildTag:  mn.BuildTag,
		Delete:    mn.Delete,
		Provider: event.Provider{
			ID:             mn.ID,
			TotalStake:     mn.TotalStaked,
			DelegateWallet: mn.Settings.DelegateWallet,
			ServiceCharge:  mn.Settings.ServiceChargeRatio,
			NumDelegates:   mn.Settings.MaxNumDelegates,
			Rewards: event.ProviderRewards{
				ProviderID:   mn.ID,
				Rewards:      mn.Reward,
				TotalRewards: mn.Reward,
			},
			LastHealthCheck: mn.LastHealthCheck,
			IsKilled:        mn.Provider.IsKilled(),
		},

		Active:        mn.Status == node.NodeStatusActive,
		CreationRound: round,
	}
}

func emitAddMiner(mn *MinerNode, balances cstate.StateContextI) {
	logging.Logger.Info("emitting add or overwrite miner event")
	balances.EmitEvent(event.TypeStats, event.TagAddMiner, mn.ID, minerNodeToMinerTable(mn, balances.GetBlock().Round))
}

func emitMinerHealthCheck(mn *MinerNode, downtime uint64, balances cstate.StateContextI) {
	data := dbs.DbHealthCheck{
		ID:              mn.ID,
		LastHealthCheck: mn.LastHealthCheck,
		Downtime:        downtime,
	}

	balances.EmitEvent(event.TypeStats, event.TagMinerHealthCheck, mn.ID, data)
}

func emitUpdateMiner(mn *MinerNode, balances cstate.StateContextI, updateStatus bool) error {

	logging.Logger.Info("emitting update miner event")

	dbUpdates := dbs.DbUpdates{
		Id: mn.ID,
		Updates: map[string]interface{}{
			"n2n_host":          mn.N2NHost,
			"host":              mn.Host,
			"port":              mn.Port,
			"path":              mn.Path,
			"public_key":        mn.PublicKey,
			"short_name":        mn.ShortName,
			"build_tag":         mn.BuildTag,
			"total_stake":       mn.TotalStaked,
			"delete":            mn.Delete,
			"delegate_wallet":   mn.Settings.DelegateWallet,
			"service_charge":    mn.Settings.ServiceChargeRatio,
			"num_delegates":     mn.Settings.MaxNumDelegates,
			"last_health_check": mn.LastHealthCheck,
		},
	}

	if updateStatus {
		dbUpdates.Updates["active"] = mn.Status == node.NodeStatusActive
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, mn.ID, dbUpdates)
	return nil
}

func emitDeleteMiner(id string, balances cstate.StateContextI) {
	logging.Logger.Info("emitting delete miner event")
	balances.EmitEvent(event.TypeStats, event.TagDeleteMiner, id, id)
}
