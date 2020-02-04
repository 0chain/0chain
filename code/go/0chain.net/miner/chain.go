package miner

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

//RoundMismatch - to indicate an error where the current round and the given round don't match
const RoundMismatch = "round_mismatch"

//ErrRoundMismatch - an error object for mismatched round error
var ErrRoundMismatch = common.NewError(RoundMismatch, "Current round number of the chain doesn't match the block generation round")

//RRSMismatch -- to indicate an error when the current round RRS is different than the block that is generated. Typically happens when timeout count changes while a block is being made
const RRSMismatch = "rrs_mismatch"

//ErrRRSMismatch - and error when rrs mismatch happens
var ErrRRSMismatch = common.NewError(RRSMismatch, "RRS for current round of the chain doesn't match the block rrs")

//RoundTimeout - to indicate an error where the round timeout has happened
const RoundTimeout = "round_timeout"

//ErrRoundTimeout - an error object for round timeout error
var ErrRoundTimeout = common.NewError(RoundTimeout, "round timed out")

var minerChain = &Chain{}

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = c
	minerChain.BlockMessageChannel = make(chan *BlockMessage, 128)
	minerChain.DKGMutex = &sync.Mutex{}
	c.SetFetchedNotarizedBlockHandler(minerChain)
	c.RoundF = MinerRoundFactory{}
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	return minerChain
}

type StartChain struct {
	datastore.IDField
	Start bool
}

var startChainEntityMetadata *datastore.EntityMetadataImpl

func (sc *StartChain) GetEntityMetadata() datastore.EntityMetadata {
	return startChainEntityMetadata
}

func StartChainProvider() datastore.Entity {
	sc := &StartChain{}
	return sc
}

func SetupStartChainEntity() {
	startChainEntityMetadata = datastore.MetadataProvider()
	startChainEntityMetadata.Name = "start_chain"
	startChainEntityMetadata.Provider = StartChainProvider
	startChainEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("start_chain", startChainEntityMetadata)
}

// MinerRoundFactory -
type MinerRoundFactory struct{}

//CreateRoundF this returns an interface{} of type *miner.Round
func (mrf MinerRoundFactory) CreateRoundF(roundNum int64) interface{} {
	mc := GetMinerChain()
	r := round.NewRound(roundNum)
	mr := mc.CreateRound(r)
	mc.AddRound(mr)

	return r
}

/*Chain - A miner chain to manage the miner activities */
type Chain struct {
	*chain.Chain
	BlockMessageChannel chan *BlockMessage
	DiscoverClients     bool
	CurrentDKG          *bls.DKG
	DKGMutex            *sync.Mutex
	ViewChangeDKG       *bls.DKG
	NextViewChange      int64
	Started             bool
}

/*GetBlockMessageChannel - get the block messages channel */
func (mc *Chain) GetBlockMessageChannel() chan *BlockMessage {
	return mc.BlockMessageChannel
}

/*SetupGenesisBlock - setup the genesis block for this chain */
func (mc *Chain) SetupGenesisBlock(hash string, magicBlock *block.MagicBlock) *block.Block {
	gr, gb := mc.GenerateGenesisBlock(hash, magicBlock)
	if gr == nil || gb == nil {
		panic("Genesis round/block canot be null")
	}
	rr, ok := gr.(*round.Round)
	if !ok {
		panic("Genesis round cannot convert to *round.Round")
	}
	mgr := mc.CreateRound(rr)
	mgr.ComputeMinerRanks(gb.MagicBlock.Miners)
	mc.AddRound(mgr)
	mc.AddGenesisBlock(gb)
	return gb
}

/*CreateRound - create a round */
func (mc *Chain) CreateRound(r *round.Round) *Round {
	var mr Round
	mr.Round = r
	mr.blocksToVerifyChannel = make(chan *block.Block, mc.NumGenerators)
	mr.verificationTickets = make(map[string]*block.BlockVerificationTicket)
	return &mr
}

/*SetLatestFinalizedBlock - Set latest finalized block */
func (mc *Chain) SetLatestFinalizedBlock(ctx context.Context, b *block.Block) {
	var r = round.NewRound(b.Round)
	mr := mc.CreateRound(r)
	mr = mc.AddRound(mr).(*Round)
	mc.SetRandomSeed(mr, b.RoundRandomSeed)
	mc.AddRoundBlock(mr, b)
	mc.AddNotarizedBlock(ctx, mr, b)
	mc.Chain.SetLatestFinalizedBlock(b)
}

func (mc *Chain) deleteTxns(txns []datastore.Entity) error {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), transactionMetadataProvider)
	defer memorystore.Close(ctx)
	return transactionMetadataProvider.GetStore().MultiDelete(ctx, transactionMetadataProvider, txns)
}

/*SetPreviousBlock - set the previous block */
func (mc *Chain) SetPreviousBlock(ctx context.Context, r round.RoundI, b *block.Block, pb *block.Block) {
	b.SetPreviousBlock(pb)
	b.RoundRandomSeed = r.GetRandomSeed()
	mc.SetRoundRank(r, b)
	b.ComputeChainWeight()
}

//GetMinerRound - get the miner's version of the round
func (mc *Chain) GetMinerRound(roundNumber int64) *Round {
	r := mc.GetRound(roundNumber)
	if r == nil {
		return nil
	}
	mr, ok := r.(*Round)
	if !ok {
		return nil
	}
	return mr
}

/*SaveClients - save clients from the block */
func (mc *Chain) SaveClients(ctx context.Context, clients []*client.Client) error {
	var err error
	clientKeys := make([]datastore.Key, len(clients))
	for idx, c := range clients {
		clientKeys[idx] = c.GetKey()
	}
	clientEntityMetadata := datastore.GetEntityMetadata("client")
	cEntities := datastore.AllocateEntities(len(clients), clientEntityMetadata)
	ctx = memorystore.WithEntityConnection(common.GetRootContext(), clientEntityMetadata)
	defer memorystore.Close(ctx)
	err = clientEntityMetadata.GetStore().MultiRead(ctx, clientEntityMetadata, clientKeys, cEntities)
	if err != nil {
		return err
	}
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
	for idx, c := range clients {
		if !datastore.IsEmpty(cEntities[idx].GetKey()) {
			continue
		}
		_, cerr := client.PutClient(ctx, c)
		if cerr != nil {
			err = cerr
		}
	}
	return err
}

func (mc *Chain) ViewChange(ctx context.Context) {
	if mc.CurrentDKG == nil || mc.CurrentDKG.StartingRound <= mc.NextViewChange {
		err := mc.UpdateMagicBlock(mc.ViewChangeMagicBlock)
		if err != nil {
			Logger.DPanic(err.Error())
		}
		mc.SetDKGSFromStore(ctx, mc.MagicBlock)
	}
}

func (mc *Chain) ChainStarted(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	timeoutCount := 0
	for true {
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			var (
				start   int
				started int
			)
			mc.Miners.ForEach(func(n *node.Node) {
				// NOTE (kostyarin): C like &arg is not so effective in Go
				//                   as argument and reply
				mc.RequestStartChain(n, &start, &started)
			})
			if start >= mc.T {
				return false
			}
			if started >= mc.T {
				return true
			}
			if timeoutCount == 20 || mc.Started {
				return mc.Started
			}
			timeoutCount++
			timer = time.NewTimer(time.Millisecond * time.Duration(mc.RoundTimeoutSofttoMin))
		}
	}
	return false
}

func StartChainRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	nodeID := r.Header.Get(node.HeaderNodeID)
	mc := GetMinerChain()
	if mc.Miners.GetNode(nodeID) == nil {
		Logger.Error("failed to send start chain", zap.Any("id", nodeID))
		return nil, common.NewError("failed to send start chain", "miner is not in active set")
	}
	round, err := strconv.Atoi(r.FormValue("round"))
	if err != nil {
		Logger.Error("failed to send start chain", zap.Any("error", err))
		return nil, err
	}
	if mc.CurrentRound != int64(round) {
		Logger.Error("failed to send start chain -- different rounds", zap.Any("current_round", mc.CurrentRound), zap.Any("requested_round", round))
		return nil, common.NewError("failed to send start chain", fmt.Sprintf("differt_rounds -- current_round: %v, requested_round: %v", mc.CurrentRound, round))
	}
	message := datastore.GetEntityMetadata("start_chain").Instance().(*StartChain)
	message.Start = !mc.Started
	message.ID = r.FormValue("round")
	return message, nil
}

/*SendDKGShare sends the generated secShare to the given node */
func (mc *Chain) RequestStartChain(n *node.Node, start, started *int) error {
	if node.Self.ID == n.ID {
		if !mc.Started {
			*start++
		} else {
			*started++
		}
		return nil
	}
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		startChain, ok := entity.(*StartChain)
		if !ok {
			err := common.NewError("invalid object", fmt.Sprintf("entity: %v", entity))
			Logger.Error("failed to request start chain", zap.Any("error", err))
			return nil, err
		}
		if startChain.Start {
			*start++
		} else {
			*started++
		}
		return startChain, nil
	}
	params := &url.Values{}
	params.Add("round", strconv.FormatInt(mc.CurrentRound, 10))
	ctx := common.GetRootContext()
	n.RequestEntityFromNode(ctx, ChainStartSender, params, handler)
	return nil
}
