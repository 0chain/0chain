package minersc

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"

	"0chain.net/smartcontract/provider"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

//go:generate msgp -io=false -tests=false -v

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Phase number.
type Phase int

// known phases
const (
	Unknown Phase = iota - 1
	Start
	Contribute
	Share
	Publish
	Wait
)

func (p Phase) String() string {
	switch p {
	case Unknown:
		return "unknown"
	case Start:
		return "start"
	case Contribute:
		return "contribute"
	case Share:
		return "share"
	case Publish:
		return "publish"
	case Wait:
		return "wait"
	default:
	}
	return fmt.Sprintf("Phase<%d>", int(p))
}

func StringToPhase(s string) Phase {
	switch s {
	case "start":
		return Start
	case "contribute":
		return Contribute
	case "share":
		return Share
	case "publish":
		return Publish
	case "wait":
		return Wait
	default:
		return Unknown
	}
}

// Pool status
const (
	ACTIVE    = "ACTIVE"
	PENDING   = "PENDING"
	DELETING  = "DELETING"
	CANDELETE = "CAN DELETE"
)

var (
	AllMinersKey         = globalKeyHash("all_miners")
	AllShardersKey       = globalKeyHash("all_sharders")
	DKGMinersKey         = globalKeyHash("dkg_miners")
	MinersMPKKey         = globalKeyHash("miners_mpk")
	MagicBlockKey        = globalKeyHash("magic_block")
	GlobalNodeKey        = globalKeyHash("global_node")
	GroupShareOrSignsKey = globalKeyHash("group_share_or_signs")
	ShardersKeepKey      = globalKeyHash("sharders_keep")
	PhaseKey             = globalKeyHash("phase")
	DeleteMinersKey      = globalKeyHash("delete_miners")
	DeleteShardersKey    = globalKeyHash("delete_sharders")
	RegisterMinersKey    = globalKeyHash("register_miners")
	RegisterShardersKey  = globalKeyHash("register_sharders")

	lockAllMiners sync.Mutex
)

type (
	phaseFunctions func(balances cstate.StateContextI, gn *GlobalNode) (
		err error)
	movePhaseFunctions func(balances cstate.StateContextI, pn *PhaseNode,
		gn *GlobalNode) error
	smartContractFunction func(t *transaction.Transaction, inputData []byte,
		gn *GlobalNode, balances cstate.StateContextI) (string, error)
)

func globalKeyHash(name string) datastore.Key {
	return ADDRESS + encryption.Hash(name)
}

func NewSimpleNodes() SimpleNodes {
	return make(map[string]*SimpleNode)
}

// not thread safe
// swagger:model SimpleNodes
type SimpleNodes map[string]*SimpleNode

// Pooler represents a pool interface
type Pooler interface {
	HasNode(id string) bool
}

func (sns SimpleNodes) reduce(limit int, xPercent float64, pmbrss int64, pmbnp Pooler) (maxNodes int) {
	var pmbNodes, newNodes, selectedNodes []*SimpleNode

	// separate previous mb miners and new miners from dkg miners list
	for _, sn := range sns {
		if pmbnp != nil && pmbnp.HasNode(sn.ID) {
			pmbNodes = append(pmbNodes, sn)
			continue
		}
		newNodes = append(newNodes, sn)
	}

	// sort pmb nodes by total stake: desc
	sort.SliceStable(pmbNodes, func(i, j int) bool {
		if pmbNodes[i].TotalStaked == pmbNodes[j].TotalStaked {
			return pmbNodes[i].ID < pmbNodes[j].ID
		}

		return pmbNodes[i].TotalStaked > pmbNodes[j].TotalStaked
	})

	// calculate max nodes count for next mb
	maxNodes = min(limit, len(sns))

	// get number of nodes from previous mb that are required to be part of next mb
	x := min(len(pmbNodes), int(math.Ceil(xPercent*float64(maxNodes))))
	y := maxNodes - x

	// select first x nodes from pmb miners
	selectedNodes = pmbNodes[:x]

	// add rest of the pmb miners into new miners list
	newNodes = append(newNodes, pmbNodes[x:]...)
	sort.SliceStable(newNodes, func(i, j int) bool {
		if newNodes[i].TotalStaked == newNodes[j].TotalStaked {
			return newNodes[i].ID < newNodes[j].ID
		}

		return newNodes[i].TotalStaked > newNodes[j].TotalStaked
	})

	if len(newNodes) <= y {
		// less than allowed nodes remaining
		selectedNodes = append(selectedNodes, newNodes...)

	} else if y > 0 {
		// more than allowed nodes remaining

		// find the range of nodes with equal stakes, start (s), end (e)
		s, e := 0, len(newNodes)
		stake := newNodes[y-1].TotalStaked
		for i, sn := range newNodes {
			if s == 0 && sn.TotalStaked == stake {
				s = i
			} else if sn.TotalStaked < stake {
				e = i
				break
			}
		}

		// select nodes that don't have equal stake
		selectedNodes = append(selectedNodes, newNodes[:s]...)

		// resolve equal stake condition by randomly selecting nodes with equal stake
		newNodes = newNodes[s:e]
		for _, j := range rand.New(rand.NewSource(pmbrss)).Perm(len(newNodes)) {
			if len(selectedNodes) < maxNodes {
				selectedNodes = append(selectedNodes, newNodes[j])
			}
		}

	}

	// update map with selected nodes
	for k := range sns {
		delete(sns, k)
	}
	for _, sn := range selectedNodes {
		sns[sn.ID] = sn
	}

	return maxNodes
}

// NodeType used in pools statistic.
type NodeType int

// known node types of the Miner SC
const (
	NodeTypeUnknown NodeType = iota // unknown (zero)
	NodeTypeMiner                   // miner node
	NodeTypeSharder                 // sharder node
)

// String converted NodeType to string.
func (nt NodeType) String() string {
	switch nt {
	case NodeTypeUnknown:
		return "unknown"
	case NodeTypeMiner:
		return "miner"
	case NodeTypeSharder:
		return "sharder"
	default:
		return fmt.Sprintf("unknown node type: %d", int(nt))
	}
}

// MarshalJSON converts NodeType to appropriate JSON
// value represented as string.
func (nt NodeType) MarshalJSON() (p []byte, err error) {
	return json.Marshal(nt.String())
}

// UnmarsalJSON converts JSON value back to NodeType.
func (nt *NodeType) UnmarshalJSON(p []byte) (err error) {
	var nts string
	if err = json.Unmarshal(p, &nts); err != nil {
		return
	}
	switch nts {
	case "unknown":
		(*nt) = NodeTypeUnknown
	case "miner":
		(*nt) = NodeTypeMiner
	case "sharder":
		(*nt) = NodeTypeSharder
	default:
		err = fmt.Errorf("unknown node type: %q", nts)
	}
	return
}

// swagger:model SimpleNode
type SimpleNode struct {
	provider.Provider
	N2NHost     string        `json:"n2n_host"`
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	Path        string        `json:"path"`
	PublicKey   string        `json:"public_key"`
	ShortName   string        `json:"short_name"`
	BuildTag    string        `json:"build_tag"`
	TotalStaked currency.Coin `json:"total_stake"`
	Delete      bool          `json:"delete"`

	// settings and statistic

	// NodeType used for delegate pools statistic.
	NodeType NodeType `json:"node_type,omitempty"`

	// LastHealthCheck used to check for active node
	LastHealthCheck common.Timestamp `json:"last_health_check"`

	// Status will be set either node.NodeStatusActive or node.NodeStatusInactive
	Status int `json:"-" msg:"-"`

	//LastSettingUpdateRound will be set to round number when settings were updated
	LastSettingUpdateRound int64 `json:"last_setting_update_round"`
}

func (smn *SimpleNode) Encode() []byte {
	buff, _ := json.Marshal(smn)
	return buff
}

func (smn *SimpleNode) Decode(input []byte) error {
	return json.Unmarshal(input, smn)
}

func (smn *SimpleNode) Validate() error {
	return validate.Struct(smn)
}

func (smn *SimpleNode) GetN2NHostKey(scAddress string) string {
	return scAddress + encryption.Hash(fmt.Sprintf("node_n2n_host_port:%s:%d", smn.N2NHost, smn.Port))
}

type ViewChangeLock struct {
	DeleteViewChangeSet bool   `json:"delete_view_change_set"`
	DeleteVC            int64  `json:"delete_after_view_change"`
	Owner               string `json:"owner"`
}

func (vcl *ViewChangeLock) IsLocked(entity interface{}) bool {
	if entity == nil {
		return false
	}
	currentVC, ok := entity.(int64)
	if ok {
		return !vcl.DeleteViewChangeSet || currentVC < vcl.DeleteVC
	}
	if currentVC == 0 {
		return false // forced unlock
	}
	return true
}

func (vcl *ViewChangeLock) LockStats(entity interface{}) []byte {
	currentVC, ok := entity.(int64)
	if ok {
		p := &poolStat{
			ViewChangeLock: vcl,
			CurrentVC:      currentVC,
			Locked:         vcl.IsLocked(currentVC),
		}
		return p.encode()
	}
	return nil
}

type poolStat struct {
	*ViewChangeLock
	CurrentVC int64 `json:"current_view_change"`
	Locked    bool  `json:"locked"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

type deletePool struct {
	ProviderType spenum.Provider `json:"provider_type,omitempty"`
	ProviderID   string          `json:"provider_id,omitempty"`
}

func (dp *deletePool) Encode() []byte {
	buff, _ := json.Marshal(dp)
	return buff
}

func (dp *deletePool) Decode(input []byte) error {
	return json.Unmarshal(input, dp)
}

// swagger:model PhaseNode
type PhaseNode struct {
	Phase        Phase `json:"phase"`
	StartRound   int64 `json:"start_round"`
	CurrentRound int64 `json:"current_round"`
	Restarts     int64 `json:"restarts"`
}

func (pn *PhaseNode) GetKey() datastore.Key {
	return PhaseKey
}

func (pn *PhaseNode) Encode() []byte {
	buff, _ := json.Marshal(pn)
	return buff
}

func (pn *PhaseNode) Decode(input []byte) error {
	return json.Unmarshal(input, pn)
}

// swagger:model DKGMinerNodes
type DKGMinerNodes struct {
	MinN     int     `json:"min_n"`
	MaxN     int     `json:"max_n"`
	TPercent float64 `json:"t_percent"`
	KPercent float64 `json:"k_percent"`

	SimpleNodes    `json:"simple_nodes"`
	T              int             `json:"t"`
	K              int             `json:"k"`
	N              int             `json:"n"`
	XPercent       float64         `json:"x_percent"`
	RevealedShares map[string]int  `json:"revealed_shares"`
	Waited         map[string]bool `json:"waited"`

	// StartRound used to filter responses from old MB where sharders comes up.
	StartRound int64 `json:"start_round"`
}

func (dkgmn *DKGMinerNodes) setConfigs(gn *GlobalNode) {
	gnb := gn.MustBase()
	dkgmn.MinN = gnb.MinN
	dkgmn.MaxN = gnb.MaxN
	dkgmn.TPercent = gnb.TPercent
	dkgmn.KPercent = gnb.KPercent
	dkgmn.XPercent = gnb.XPercent
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// The min_n is checked before the calculateTKN call, so, the n >= min_n.
// The calculateTKN used to set initial T, K, and N.
func (dkgmn *DKGMinerNodes) calculateTKN(gn *GlobalNode, n int) {
	dkgmn.setConfigs(gn)
	var m = min(dkgmn.MaxN, n)
	dkgmn.N = m
	dkgmn.K = int(math.Ceil(dkgmn.KPercent * float64(m)))
	dkgmn.T = int(math.Ceil(dkgmn.TPercent * float64(m)))
}

func simpleNodesKeys(sns SimpleNodes) (ks []string) {
	ks = make([]string, 0, len(sns))
	for k := range sns {
		ks = append(ks, k)
	}
	return
}

// reduce method checks boundaries and if final, reduces the
// list to adhere to the limits (min_n, max_n) and conditions
func (dkgmn *DKGMinerNodes) reduceNodes(
	final bool,
	gn *GlobalNode,
	balances cstate.StateContextI) (err error) {

	var n = len(dkgmn.SimpleNodes)

	if n < dkgmn.MinN {
		return fmt.Errorf("too few miners: %d, want at least: %d", n, dkgmn.MinN)
	}

	if !gn.hasPrevDKGMiner(dkgmn.SimpleNodes, balances) {
		return fmt.Errorf("missing miner from previous set, n: %d, list: %s",
			n, simpleNodesKeys(dkgmn.SimpleNodes))
	}

	if final {
		simpleNodes := make(SimpleNodes)
		for k, v := range dkgmn.SimpleNodes {
			simpleNodes[k] = v
		}
		var pmbrss int64
		var pmbnp *node.Pool
		pmb := balances.GetLastestFinalizedMagicBlock()
		if pmb != nil {
			pmbrss = pmb.RoundRandomSeed
			if pmb.MagicBlock != nil {
				pmbnp = pmb.MagicBlock.Miners
			}
		}
		gnb := gn.MustBase()
		simpleNodes.reduce(gnb.MaxN, gnb.XPercent, pmbrss, pmbnp)
		dkgmn.SimpleNodes = simpleNodes
	}

	return
}

func NewDKGMinerNodes() *DKGMinerNodes {
	return &DKGMinerNodes{
		SimpleNodes:    NewSimpleNodes(),
		RevealedShares: make(map[string]int),
		Waited:         make(map[string]bool),
	}
}

func (dmn *DKGMinerNodes) Encode() []byte {
	buff, _ := json.Marshal(dmn)
	return buff
}

func (dmn *DKGMinerNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, dmn)
	if err != nil {
		return err
	}
	return nil
}

func (dmn *DKGMinerNodes) GetHash() string {
	return util.ToHex(dmn.GetHashBytes())
}

func (dmn *DKGMinerNodes) GetHashBytes() []byte {
	return encryption.RawHash(dmn.Encode())
}

// getMinersList returns miners list
func getMinersList(state cstate.StateContextI) (*MinerNodes, error) {
	minerNodes, err := getNodesList(getMinerNode, state, AllMinersKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		return &MinerNodes{}, nil
	}

	return minerNodes, nil
}

//nolint:unused
func updateMinersList(state cstate.StateContextI, miners *MinerNodes) error {
	nodeIDs := make(NodeIDs, len(miners.Nodes))
	for i, m := range miners.Nodes {
		nodeIDs[i] = m.ID
	}
	if _, err := state.InsertTrieNode(AllMinersKey, &nodeIDs); err != nil {
		return common.NewError("update_all_miners_list_failed", err.Error())
	}
	return nil
}

// getDKGMinersList gets dkg miners list
func getDKGMinersList(state cstate.CommonStateContextI) (*DKGMinerNodes, error) {
	dkgMiners := NewDKGMinerNodes()
	err := state.GetTrieNode(DKGMinersKey, dkgMiners)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		logging.Logger.Debug("VC: no dkg miners list found, create one")

		return NewDKGMinerNodes(), nil
	}

	return dkgMiners, nil
}

// updateDKGMinersList update the dkg miners list
func updateDKGMinersList(state cstate.StateContextI, dkgMiners *DKGMinerNodes) error {
	logging.Logger.Info("update dkg miners list", zap.Int("len", len(dkgMiners.SimpleNodes)))
	_, err := state.InsertTrieNode(DKGMinersKey, dkgMiners)
	return err
}

func getMinersMPKs(state cstate.CommonStateContextI) (*block.Mpks, error) {
	mpks := block.NewMpks()
	err := state.GetTrieNode(MinersMPKKey, mpks)
	if err != nil {
		return nil, err
	}

	return mpks, nil
}

func updateMinersMPKs(state cstate.StateContextI, mpks *block.Mpks) error {
	_, err := state.InsertTrieNode(MinersMPKKey, mpks)
	return err
}

func getMagicBlock(state cstate.CommonStateContextI) (*block.MagicBlock, error) {
	magicBlock := block.NewMagicBlock()
	err := state.GetTrieNode(MagicBlockKey, magicBlock)
	if err != nil {
		return nil, err
	}
	logging.Logger.Debug("get magic block", zap.Any("magic block", magicBlock))

	return magicBlock, nil
}

func updateMagicBlock(state cstate.StateContextI, magicBlock *block.MagicBlock) error {
	logging.Logger.Debug("save magic block", zap.Any("magic block", magicBlock))
	_, err := state.InsertTrieNode(MagicBlockKey, magicBlock)
	return err
}

func getGroupShareOrSigns(state cstate.CommonStateContextI) (*block.GroupSharesOrSigns, error) {
	var gsos = block.NewGroupSharesOrSigns()
	err := state.GetTrieNode(GroupShareOrSignsKey, gsos)
	if err != nil {
		return nil, err
	}

	return gsos, nil
}

func updateGroupShareOrSigns(state cstate.StateContextI, gsos *block.GroupSharesOrSigns) error {
	_, err := state.InsertTrieNode(GroupShareOrSignsKey, gsos)
	return err
}

// getShardersKeepList returns the sharder list
func getShardersKeepList(balances cstate.StateContextI) (*MinerNodes, error) {
	sharders, err := getNodesList(getSharderNode, balances, ShardersKeepKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return &MinerNodes{}, nil
	}

	return sharders, nil
}

func updateShardersKeepList(state cstate.StateContextI, nodeIDs NodeIDs) error {
	_, err := state.InsertTrieNode(ShardersKeepKey, &nodeIDs)
	return err
}

// getAllShardersKeepList returns the sharder list
func getAllShardersList(balances cstate.StateContextI) (*MinerNodes, error) {
	sharders, err := getNodesList(getSharderNode, balances, AllShardersKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return &MinerNodes{}, nil
	}
	return sharders, nil
}

//nolint:unused
func updateAllShardersList(state cstate.StateContextI, sharders *MinerNodes) error {
	nodeIDs := make(NodeIDs, len(sharders.Nodes))
	for i, n := range sharders.Nodes {
		nodeIDs[i] = n.ID
	}

	_, err := state.InsertTrieNode(AllShardersKey, &nodeIDs)
	return err
}

// NodeIDs stores all the node IDs for miners or sharders
// We will refactor to store it to partitions later, but for now, it should be fine
// to store in a single MPT node as the data size is small.
type NodeIDs []string

func getNodeIDs(state cstate.CommonStateContextI, key string) (NodeIDs, error) {
	var nIDs NodeIDs
	err := state.GetTrieNode(key, &nIDs)
	switch err {
	case nil:
		return nIDs, nil
	case util.ErrValueNotPresent:
		return NodeIDs{}, nil
	default:
		return nil, err
	}
}

func (n *NodeIDs) save(state cstate.StateContextI, key string) error {
	_, err := state.InsertTrieNode(key, n)
	return err
}

func (n *NodeIDs) find(id string) bool {
	for _, nID := range *n {
		if nID == id {
			return true
		}
	}
	return false
}

func getNodesList(
	getNode func(id string, state cstate.StateContextI) (*MinerNode, error),
	balances cstate.StateContextI,
	key datastore.Key,
) (*MinerNodes, error) {
	nIDs, err := getNodeIDs(balances, key)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(nIDs))
	for _, id := range nIDs {
		ids = append(ids, id)
	}

	ss, err := cstate.GetItemsByIDs(ids, getNode, balances)
	if err != nil {
		return nil, err
	}

	return &MinerNodes{ss}, nil
}

// quick fix: localhost check + duplicate check
// TODO: remove this after more robust challenge based node addtion/health_check is added
//
//nolint:unused
func quickFixDuplicateHosts(nn *MinerNode, allNodes []*MinerNode) error {
	localhost := regexp.MustCompile(`^(?:(?:https|http)\:\/\/)?(?:localhost|127\.0\.0\.1)(?:\:\d+)?(?:\/.*)?$`)
	host := strings.TrimSpace(nn.Host)
	n2nhost := strings.TrimSpace(nn.N2NHost)
	port := nn.Port
	if n2nhost == "" || localhost.MatchString(n2nhost) {
		return fmt.Errorf("invalid n2nhost: '%v'", n2nhost)
	}
	if host == "" || localhost.MatchString(host) {
		host = n2nhost
	}
	for _, n := range allNodes {
		if n.ID != nn.ID && n2nhost == n.N2NHost && n.Port == port {
			return fmt.Errorf("n2nhost:port already exists: '%v:%v'", n2nhost, port)
		}
		if n.ID != nn.ID && host == n.Host && n.Port == port {
			return fmt.Errorf("host:port already exists: '%v:%v'", host, port)
		}
	}
	nn.Host, nn.N2NHost, nn.Port = host, n2nhost, port
	return nil
}

func insertNodeN2NHost(balances cstate.StateContextI, scAddress string, nn *MinerNode) error {
	localhost := regexp.MustCompile(`^(?:(?:https|http)\:\/\/)?(?:localhost|127\.0\.0\.1)(?:\:\d+)?(?:\/.*)?$`)
	host := strings.TrimSpace(nn.Host)
	n2nhost := strings.TrimSpace(nn.N2NHost)
	port := nn.Port
	if n2nhost == "" || localhost.MatchString(n2nhost) {
		return fmt.Errorf("invalid n2nhost: '%v'", n2nhost)
	}
	if host == "" || localhost.MatchString(host) {
		host = n2nhost
	}

	nn.Host, nn.N2NHost, nn.Port = host, n2nhost, port
	key := nn.GetN2NHostKey(scAddress)
	err := balances.GetTrieNode(key, &datastore.NOIDField{})
	switch err {
	case nil:
		return fmt.Errorf("n2nhost:port already exists: '%v:%v'", n2nhost, port)
	case util.ErrValueNotPresent:
		_, err = balances.InsertTrieNode(key, &datastore.NOIDField{})
		if err != nil {
			return fmt.Errorf("insert node n2nhost:port failed: %v", err)
		}
		return nil
	default:
		return err
	}
}
