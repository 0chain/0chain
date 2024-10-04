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
	if err := kill(input, txn.ClientID, gn.MustBase().OwnerId, getMinerNode, balances); err != nil {
		return "", common.NewError("kill_miner_failed", err.Error())
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
	if err := kill(input, txn.ClientID, gn.MustBase().OwnerId, getSharderNode, balances); err != nil {
		return "", common.NewError("kill_sharder_failed", err.Error())
	}
	return "", nil
}

// kill
// kills a miner or sharder. We do not use Provider.kill() as that will also slash the stake pools.
func kill(
	input []byte,
	clientId, ownerId string,
	getNode func(string, cstate.StateContextI) (*MinerNode, error),
	balances cstate.StateContextI,
) error {
	var req provider.ProviderRequest
	if err := req.Decode(input); err != nil {
		return err
	}

	if err := smartcontractinterface.AuthorizeWithOwner("only the owner can kill a provider", func() bool {
		return ownerId == clientId
	}); err != nil {
		return err
	}

	node, err := getNode(req.ID, balances)
	if err != nil {
		return err
	}

	if node.SimpleNode.HasBeenKilled && node.StakePool.HasBeenKilled {
		return fmt.Errorf("%s is already killed", req.ID)
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
