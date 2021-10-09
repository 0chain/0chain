package convert

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	minerproto "0chain.net/miner/proto/api/src/proto"
)

func BlockToGRPCBlock(b *block.Block) *minerproto.Block {
	if b == nil {
		return nil
	}

	return &minerproto.Block{
		VerificationTickets:            VerificationTicketsToGRPCVerificationTickets(b.VerificationTickets),
		Hash:                           b.Hash,
		Signature:                      b.Signature,
		ChainId:                        b.ChainID,
		ChainWeight:                    b.ChainWeight,
		RunningTxnCount:                b.RunningTxnCount,
		MagicBlock:                     MagicBlockToGRPCMagicBlock(b.MagicBlock),
		Version:                        b.Version,
		CreationDate:                   int64(b.CreationDate),
		LatestFinalizedMagicBlockHash:  b.LatestFinalizedMagicBlockHash,
		LatestFinalizedMagicBlockRound: b.LatestFinalizedMagicBlockRound,
		PrevHash:                       b.PrevHash,
		PrevVerificationTickets:        VerificationTicketsToGRPCVerificationTickets(b.PrevBlockVerificationTickets),
		MinerId:                        b.MinerID,
		Round:                          b.Round,
		RoundRandomSeed:                b.RoundRandomSeed,
		RoundTimeoutCount:              int64(b.RoundTimeoutCount),
		StateHash:                      string(b.ClientStateHash),
		Transactions:                   TransactionsToGRPCTransactions(b.Txns),
	}
}

func VerificationTicketsToGRPCVerificationTickets(tickets []*block.VerificationTicket) []*minerproto.VerificationTicket {
	if tickets == nil {
		return nil
	}

	res := make([]*minerproto.VerificationTicket, len(tickets))
	for idx, ticket := range tickets {
		res[idx] = &minerproto.VerificationTicket{
			VerifierId: ticket.VerifierID,
			Signature:  ticket.Signature,
		}
	}
	return res
}

func MagicBlockToGRPCMagicBlock(magicBlock *block.MagicBlock) *minerproto.MagicBlock {
	if magicBlock == nil {
		return nil
	}

	return &minerproto.MagicBlock{
		Hash:             magicBlock.Hash,
		PreviousHash:     magicBlock.PreviousMagicBlockHash,
		MagicBlockNumber: magicBlock.MagicBlockNumber,
		StartingRound:    magicBlock.StartingRound,
		Miners:           PoolToGRPCPool(magicBlock.Miners),
		Sharders:         PoolToGRPCPool(magicBlock.Sharders),
		Shares:           nil,
		Mpks:             nil,
		T:                0,
		K:                0,
		N:                0,
	}
}

func PoolToGRPCPool(pool *node.Pool) *minerproto.Pool {
	if pool == nil {
		return nil
	}

	nodes := make([]*minerproto.Pool_Node, len(pool.Nodes))
	for idx, n := range pool.Nodes {
		nodes[idx] = &minerproto.Pool_Node{
			Id:           n.ID,
			Version:      n.Version,
			CreationDate: int64(n.CreationDate),
			PublicKey:    n.PublicKey,
			N2NHost:      n.N2NHost,
			Host:         n.Host,
			Port:         int64(n.Port),
			GrpcPort:     int64(n.GRPCPort),
			Path:         n.Path,
			Type:         int64(n.Type),
			Description:  n.Description,
			SetIndex:     int64(n.SetIndex),
			Status:       int64(n.Status),
			InPrevMb:     n.InPrevMB,
			Info: &minerproto.Pool_Node_Info{
				BuildTag:                n.Info.BuildTag,
				StateMissingNodes:       n.Info.StateMissingNodes,
				MinersMedianNetworkTime: int64(n.Info.MinersMedianNetworkTime),
				AvgBlockTxns:            int64(n.Info.AvgBlockTxns),
			},
		}
	}

	return &minerproto.Pool{
		Type:  int64(pool.Type),
		Nodes: nil,
	}
}

func TransactionsToGRPCTransactions(txns []*transaction.Transaction) []*minerproto.Transaction {
	if txns == nil {
		return nil
	}

	res := make([]*minerproto.Transaction, len(txns))
	for idx, txn := range txns {
		res[idx] = &minerproto.Transaction{
			Hash:              txn.Hash,
			Version:           txn.Version,
			ClientId:          txn.ClientID,
			ToClientId:        txn.ToClientID,
			ChainId:           txn.ChainID,
			TransactionData:   txn.TransactionData,
			TransactionValue:  txn.Value,
			Signature:         txn.Signature,
			CreationDate:      int64(txn.CreationDate),
			TransactionFee:    txn.Fee,
			TransactionType:   int64(txn.TransactionType),
			TransactionOutput: txn.TransactionOutput,
			TxnOutputHash:     txn.OutputHash,
			TransactionStatus: int64(txn.Status),
		}
	}
	return res
}
