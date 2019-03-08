package smartcontract

import (
	"strconv"
	"0chain.net/chaincore/transaction"
	"0chain.net/chaincore/node"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
) 

const (
	Miner = "MINER"
	Sharder = "SHARDER"
)

//BCContext a wrapper to access Blockchain
type BCContext struct{}

// PoolMemberInfo of pool members
type PoolMemberInfo struct {
	N2NHost        string `json:"n2n_host"`
	PublicKey           string `json:"public_key"`
	Port           string `json:"port"`
	Type           string `json:"type"`
}

type PoolMembersInfo struct {
	MembersInfo   []PoolMemberInfo `json:"members_info"`
}

func (bc *BCContext) GetNodepoolInfo() interface{} {
	Logger.Info("Here inside GetNodepool", zap.Any("x", transaction.TXN_TIME_TOLERANCE))

	//blockchain := chain.GetServerChain()
	nodes := node.GetNodes()
	members := &PoolMembersInfo{}
	members.MembersInfo = make([]PoolMemberInfo, 0, len(nodes))

	

	for _, n := range nodes {
		pm := &PoolMemberInfo{}
		pm.N2NHost=n.N2NHost
		pm.Port=strconv.Itoa(n.Port)
		switch n.Type { 
			case node.NodeTypeMiner:
				pm.Type = Miner
			case node.NodeTypeSharder:
				pm.Type = Sharder
			default:
				Logger.Info("unknown_node_type", zap.Int8("Type", n.Type))
		}
		pm.PublicKey=n.PublicKey
		Logger.Info("Adding poolmember ", zap.String("Type", pm.Type), 
					zap.String("N2nHost", pm.N2NHost))
		members.MembersInfo = append(members.MembersInfo, *pm)

	}
	Logger.Info("GetNodePoolInfo returning ", zap.Int("membersInfo", len(members.MembersInfo)))
	return members
}
