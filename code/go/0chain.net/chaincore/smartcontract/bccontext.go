package smartcontract

import (
	"strconv"

	"0chain.net/chaincore/node"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
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
		typename, err := node.GetNodeTypeName(n)
		if err != nil {
			logging.Logger.Info(err.Error())
		} else {
			pm.Type = typename
		}
		pm.PublicKey = n.PublicKey
		//Logger.Info("Adding poolmember ", zap.String("Type", pm.Type), zap.String("N2nHost", pm.N2NHost))
		members.MembersInfo = append(members.MembersInfo, *pm)
	}
	logging.Logger.Info("GetNodePoolInfo returning ", zap.Int("membersInfo", len(members.MembersInfo)))
	return members
}
