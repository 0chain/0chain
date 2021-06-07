package chain

import (
	"context"
	"time"

	"github.com/0chain/0chain/code/go/0chain.net/core/common"

	"github.com/0chain/0chain/code/go/0chain.net/core/datastore"

	"github.com/0chain/0chain/code/go/0chain.net/core/util"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/threshold/bls"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/client"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/node"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/block"

	"github.com/0chain/0chain/code/go/0chain.net/miner/minerGRPC"
)

func NewGRPCMinerChainService(chain IChain) *minerChainGRPCService {
	return &minerChainGRPCService{
		ServerChain: chain,
	}
}

type IChain interface {
	GetLatestFinalizedBlockSummary() *block.BlockSummary
}

type minerChainGRPCService struct {
	ServerChain IChain
}

func (m *minerChainGRPCService) GetLatestFinalizedBlockSummary(ctx context.Context, req *minerGRPC.GetLatestFinalizedBlockSummaryRequest) (*minerGRPC.GetLatestFinalizedBlockSummaryResponse, error) {
	return &minerGRPC.GetLatestFinalizedBlockSummaryResponse{
		BlockSummary: BlockSummaryToGRPCBlockSummary(m.ServerChain.GetLatestFinalizedBlockSummary()),
	}, nil
}

func BlockSummaryToGRPCBlockSummary(summary *block.BlockSummary) *minerGRPC.BlockSummary {
	if summary == nil {
		return nil
	}

	return &minerGRPC.BlockSummary{
		Hash:                  summary.Hash,
		MinerId:               summary.MinerID,
		Round:                 summary.Round,
		RoundRandomSeed:       summary.RoundRandomSeed,
		MerkleTreeRoot:        summary.MerkleTreeRoot,
		ClientStateHash:       string(summary.ClientStateHash),
		ReceiptMerkleTreeRoot: summary.ReceiptMerkleTreeRoot,
		NumTxns:               int64(summary.NumTxns),
		Version:               summary.Version,
		CreationDate:          int64(summary.CreationDate),
		MagicBlock:            MagicBlockToMagicBlockGRPC(summary.MagicBlock),
	}
}

func ShareOrSignsToGRPCShareOrSigns(shareOrSigns *block.ShareOrSigns) *minerGRPC.ShareOrSigns {
	if shareOrSigns == nil {
		return nil
	}

	var shareOrSign []*minerGRPC.StringMapDKGKeyShare
	for k, v := range shareOrSigns.ShareOrSigns {
		shareOrSign = append(shareOrSign, &minerGRPC.StringMapDKGKeyShare{
			Key:         k,
			DkgKeyShare: DKGKeyShareToGRPCDKGKeyShare(v),
		})
	}

	return &minerGRPC.ShareOrSigns{
		Id:          shareOrSigns.ID,
		ShareOrSign: shareOrSign,
	}
}

func DKGKeyShareToGRPCDKGKeyShare(dkgKeyShare *bls.DKGKeyShare) *minerGRPC.DKGKeyShare {
	if dkgKeyShare == nil {
		return nil
	}

	return &minerGRPC.DKGKeyShare{
		Id:      dkgKeyShare.ID,
		Message: dkgKeyShare.Message,
		Share:   dkgKeyShare.Share,
		Sign:    dkgKeyShare.Sign,
	}
}

func MPKToGRPCMPK(mpk *block.MPK) *minerGRPC.MPK {
	if mpk == nil {
		return nil
	}

	return &minerGRPC.MPK{
		Id:  mpk.ID,
		Mpk: mpk.Mpk,
	}
}

func MagicBlockToMagicBlockGRPC(b *block.MagicBlock) *minerGRPC.MagicBlock {
	if b == nil {
		return nil
	}

	var shareOrSigns []*minerGRPC.StringMapShareOrSigns
	for k, v := range b.ShareOrSigns.Shares {
		shareOrSigns = append(shareOrSigns, &minerGRPC.StringMapShareOrSigns{
			Key:   k,
			Share: ShareOrSignsToGRPCShareOrSigns(v),
		})
	}

	var mpks []*minerGRPC.StringMapMPK
	for k, v := range b.Mpks.Mpks {
		mpks = append(mpks, &minerGRPC.StringMapMPK{
			Key: k,
			Mpk: MPKToGRPCMPK(v),
		})
	}

	return &minerGRPC.MagicBlock{
		Hash:                   b.Hash,
		PreviousMagicBlockHash: b.PreviousMagicBlockHash,
		MagicBlockNumber:       b.MagicBlockNumber,
		StartingRound:          b.StartingRound,
		Miners:                 NodePoolToGRPCNodePool(b.Miners),
		Sharders:               NodePoolToGRPCNodePool(b.Sharders),
		ShareOrSigns:           shareOrSigns,
		Mpks:                   mpks,
		T:                      int64(b.T),
		K:                      int64(b.K),
		N:                      int64(b.N),
	}
}

func NodeToGRPCNode(node *node.Node) *minerGRPC.Node {
	if node == nil {
		return nil
	}

	return &minerGRPC.Node{
		Client:      ClientToGRPCClient(&node.Client),
		N2NHost:     node.N2NHost,
		Host:        node.Host,
		Port:        int64(node.Port),
		Path:        node.Path,
		Type:        int64(node.Type),
		Description: node.Description,
		SetIndex:    int64(node.SetIndex),
		Status:      int64(node.Status),
		Info:        InfoToGRPCInfo(&node.Info),
	}
}

func ClientToGRPCClient(client *client.Client) *minerGRPC.Client {
	if client == nil {
		return nil
	}

	return &minerGRPC.Client{
		Id:           client.ID,
		Version:      client.Version,
		CreationDate: int64(client.CreationDate),
		PubKey:       client.PublicKey,
	}
}

func InfoToGRPCInfo(info *node.Info) *minerGRPC.Info {
	if info == nil {
		return nil
	}

	return &minerGRPC.Info{
		BuildTag:                info.BuildTag,
		StateMissingNodes:       info.StateMissingNodes,
		MinersMedianNetworkTime: int64(info.MinersMedianNetworkTime),
		AvgBlockTxns:            int64(info.AvgBlockTxns),
	}
}

func NodePoolToGRPCNodePool(pool *node.Pool) *minerGRPC.NodePool {
	if pool == nil {
		return nil
	}

	var nodes []*minerGRPC.StringMapNode
	for k, v := range pool.NodesMap {
		nodes = append(nodes, &minerGRPC.StringMapNode{
			Key:  k,
			Node: NodeToGRPCNode(v),
		})
	}

	return &minerGRPC.NodePool{
		Type:  int64(pool.Type),
		Nodes: nodes,
	}
}

func BlockSummaryGRPCToBlockSummary(summary *minerGRPC.BlockSummary) *block.BlockSummary {
	if summary == nil {
		return nil
	}

	return &block.BlockSummary{
		Hash:                  summary.Hash,
		MinerID:               summary.MinerId,
		Round:                 summary.Round,
		RoundRandomSeed:       summary.RoundRandomSeed,
		MerkleTreeRoot:        summary.MerkleTreeRoot,
		ClientStateHash:       util.Key(summary.ClientStateHash),
		ReceiptMerkleTreeRoot: summary.ReceiptMerkleTreeRoot,
		NumTxns:               int(summary.NumTxns),
		VersionField:          datastore.VersionField{Version: summary.Version},
		CreationDateField:     datastore.CreationDateField{CreationDate: common.Timestamp(summary.CreationDate)},
		MagicBlock:            MagicBlockGRPCToMagicBlock(summary.MagicBlock),
	}
}

func ShareOrSignsGRPCToShareOrSigns(shareOrSigns *minerGRPC.ShareOrSigns) *block.ShareOrSigns {
	if shareOrSigns == nil {
		return nil
	}

	var shareOrSign = make(map[string]*bls.DKGKeyShare)
	for _, v := range shareOrSigns.ShareOrSign {
		shareOrSign[v.Key] = DKGKeyShareGRPCToDKGKeyShare(v.DkgKeyShare)
	}

	return &block.ShareOrSigns{
		ID:           shareOrSigns.Id,
		ShareOrSigns: shareOrSign,
	}
}

func DKGKeyShareGRPCToDKGKeyShare(dkgKeyShare *minerGRPC.DKGKeyShare) *bls.DKGKeyShare {
	if dkgKeyShare == nil {
		return nil
	}

	return &bls.DKGKeyShare{
		IDField: datastore.IDField{ID: dkgKeyShare.Id},
		Message: dkgKeyShare.Message,
		Share:   dkgKeyShare.Share,
		Sign:    dkgKeyShare.Sign,
	}
}

func MPKGRPCToMPK(mpk *minerGRPC.MPK) *block.MPK {
	if mpk == nil {
		return nil
	}

	return &block.MPK{
		ID:  mpk.Id,
		Mpk: mpk.Mpk,
	}
}

func MagicBlockGRPCToMagicBlock(b *minerGRPC.MagicBlock) *block.MagicBlock {
	if b == nil {
		return nil
	}

	var shareOrSigns = make(map[string]*block.ShareOrSigns)
	for _, v := range b.ShareOrSigns {
		shareOrSigns[v.Key] = ShareOrSignsGRPCToShareOrSigns(v.Share)
	}

	var mpks = make(map[string]*block.MPK)
	for _, v := range b.Mpks {
		mpks[v.Key] = MPKGRPCToMPK(v.Mpk)
	}

	return &block.MagicBlock{
		HashIDField:            datastore.HashIDField{Hash: b.Hash},
		PreviousMagicBlockHash: b.PreviousMagicBlockHash,
		MagicBlockNumber:       b.MagicBlockNumber,
		StartingRound:          b.StartingRound,
		Miners:                 NodePoolGRPCToNodePool(b.Miners),
		Sharders:               NodePoolGRPCToNodePool(b.Sharders),
		ShareOrSigns:           &block.GroupSharesOrSigns{Shares: shareOrSigns},
		Mpks:                   &block.Mpks{Mpks: mpks},
		T:                      int(b.T),
		K:                      int(b.K),
		N:                      int(b.N),
	}
}

func NodeGRPCToNode(n *minerGRPC.Node) *node.Node {
	if n == nil {
		return nil
	}

	return &node.Node{
		Client:      *ClientGRPCToClient(n.Client),
		N2NHost:     n.N2NHost,
		Host:        n.Host,
		Port:        int(n.Port),
		Path:        n.Path,
		Type:        int8(n.Type),
		Description: n.Description,
		SetIndex:    int(n.SetIndex),
		Status:      int(n.Status),
		Info:        *InfoGRPCToInfo(n.Info),
	}
}

func ClientGRPCToClient(c *minerGRPC.Client) *client.Client {
	if c == nil {
		return nil
	}

	return &client.Client{
		IDField:           datastore.IDField{ID: c.Id},
		VersionField:      datastore.VersionField{Version: c.Version},
		CreationDateField: datastore.CreationDateField{CreationDate: common.Timestamp(c.CreationDate)},
		PublicKey:         c.PubKey,
	}
}

func InfoGRPCToInfo(info *minerGRPC.Info) *node.Info {
	if info == nil {
		return nil
	}

	return &node.Info{
		BuildTag:                info.BuildTag,
		StateMissingNodes:       info.StateMissingNodes,
		MinersMedianNetworkTime: time.Duration(info.MinersMedianNetworkTime),
		AvgBlockTxns:            int(info.AvgBlockTxns),
	}
}

func NodePoolGRPCToNodePool(pool *minerGRPC.NodePool) *node.Pool {
	if pool == nil {
		return nil
	}

	var nodes = make(map[string]*node.Node)
	for _, v := range pool.Nodes {
		nodes[v.Key] = NodeGRPCToNode(v.Node)
	}

	return &node.Pool{
		Type:     int8(pool.Type),
		NodesMap: nodes,
	}
}
