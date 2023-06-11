package minersc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/provider"
)

// killMiner
// killing is permanent and a killed miner cannot receive any rewards
func (_ *MinerSmartContract) killMiner(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	var req provider.ProviderRequest
	if err := req.Decode(input); err != nil {
		return "", common.NewErrorf("kill_sharder_failed", "decoding request: %v", err)
	}
	if err := kill(req.ID, txn.ClientID, gn.OwnerId, getMinerNode, balances); err != nil {
		return "", common.NewError("kill_miner_failed", err.Error())
	}

	if mb := balances.GetChainCurrentMagicBlock(); mb != nil {
		for _, nd := range mb.Miners.Nodes {
			if nd.GetKey() == req.ID {
				nd.SetKilled(true)
			}
		}
	}

	return "", nil
}

// killSharder
// killing is permanent and a killed miner cannot receive any rewards
func (_ *MinerSmartContract) killSharder(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	var req provider.ProviderRequest
	if err := req.Decode(input); err != nil {
		return "", common.NewErrorf("kill_sharder_failed", "decoding request: %v", err)
	}

	if err := kill(req.ID, txn.ClientID, gn.OwnerId, getSharderNode, balances); err != nil {
		return "", common.NewError("kill_sharder_failed", err.Error())
	}

	if mb := balances.GetChainCurrentMagicBlock(); mb != nil {
		for _, nd := range mb.Sharders.Nodes {
			if nd.GetKey() == req.ID {
				nd.SetKilled(true)
			}
		}
	}

	return "", nil
}

// kill
// kills a miner or sharder. We do not use Provider.kill() as that will also slash the stake pools.
func kill(
	id string,
	clientId, ownerId string,
	getNode func(string, cstate.CommonStateContextI) (*MinerNode, error),
	balances cstate.StateContextI,
) error {
	if err := smartcontractinterface.AuthorizeWithOwner("only the owner can kill a provider", func() bool {
		return ownerId == clientId
	}); err != nil {
		return err
	}

	node, err := getNode(id, balances)
	if err != nil {
		return err
	}

	if node.SimpleNode.HasBeenKilled && node.StakePool.HasBeenKilled {
		return fmt.Errorf("%s is already killed", id)
	}

	node.SimpleNode.HasBeenKilled = true
	node.StakePool.HasBeenKilled = true

	if err := node.save(balances); err != nil {
		return err
	}

	balances.EmitEvent(event.TypeStats, event.TagKillProvider, node.Id(), dbs.ProviderID{
		ID:   node.Id(),
		Type: node.Type(),
	})

	return nil
}
