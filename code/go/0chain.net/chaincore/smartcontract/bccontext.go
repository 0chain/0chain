package smartcontract

import (
	"strconv"

	"0chain.net/chaincore/node"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

const (
	//Miner member type
	Miner = "MINER"
	//Sharder member type
	Sharder = "SHARDER"
)

//BCContext a wrapper to access Blockchain
type BCContext struct{}

// PoolMemberInfo of a pool member
type PoolMemberInfo struct {
	N2NHost   string `json:"n2n_host"`
	PublicKey string `json:"public_key"`
	Port      string `json:"port"`
	Type      string `json:"type"`
}

//PoolMembersInfo array of pool memebers
// swagger:model PoolMembersInfo
type PoolMembersInfo struct {
	MembersInfo []PoolMemberInfo `json:"members_info"`
}

/*GetNodepoolInfo gets complete information about node pool members.
  Smartcontracts using this must have validated the caller.
*/
func (bc *BCContext) GetNodepoolInfo() interface{} {
	nodes := node.CopyNodes()
	members := &PoolMembersInfo{}
	members.MembersInfo = make([]PoolMemberInfo, 0, len(nodes))

	for _, n := range nodes {
		pm := &PoolMemberInfo{}
		pm.N2NHost = n.N2NHost
		pm.Port = strconv.Itoa(n.Port)
		switch n.Type {
		case node.NodeTypeMiner:
			pm.Type = Miner
		case node.NodeTypeSharder:
			pm.Type = Sharder
		default:
			logging.Logger.Info("unknown_node_type", zap.Int8("Type", int8(n.Type)))
		}
		pm.PublicKey = n.PublicKey
		//Logger.Info("Adding poolmember ", zap.String("Type", pm.Type), zap.String("N2nHost", pm.N2NHost))
		members.MembersInfo = append(members.MembersInfo, *pm)
	}
	logging.Logger.Info("GetNodePoolInfo returning ", zap.Int("membersInfo", len(members.MembersInfo)))
	return members
}
