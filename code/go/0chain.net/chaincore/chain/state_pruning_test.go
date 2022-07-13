package chain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingNodeDB struct {
	underlying util.NodeDB
}

/*GetNode - implement interface */
func (fndb *failingNodeDB) GetNode(key util.Key) (util.Node, error) {
	return fndb.underlying.GetNode(key)
}

/*PutNode - implement interface */
func (fndb *failingNodeDB) PutNode(key util.Key, node util.Node) error {
	return fndb.underlying.PutNode(key, node)
}

/*DeleteNode - implement interface */
func (fndb *failingNodeDB) DeleteNode(key util.Key) error {
	return fndb.underlying.DeleteNode(key)
}

/*MultiGetNode - get multiple nodes */
func (fndb *failingNodeDB) MultiGetNode(keys []util.Key) (nodes []util.Node, err error) {
	return fndb.underlying.MultiGetNode(keys)
}

/*MultiPutNode - implement interface */
func (fndb *failingNodeDB) MultiPutNode(keys []util.Key, nodes []util.Node) error {
	return errors.New("some error")
}

/*MultiDeleteNode - implement interface */
func (fndb *failingNodeDB) MultiDeleteNode(keys []util.Key) error {
	return fndb.underlying.MultiDeleteNode(keys)
}

/*Iterate - implement interface */
func (fndb *failingNodeDB) Iterate(ctx context.Context, handler util.NodeDBIteratorHandler) error {
	return fndb.underlying.Iterate(ctx, handler)
}

/*Size - implement interface */
func (fndb *failingNodeDB) Size(ctx context.Context) int64 {
	return fndb.underlying.Size(ctx)
}

func Test_pruneClientState_withFailingMutliPutNode(t *testing.T) {
	db, err := util.NewPNodeDB("/tmp/mpt", "/tmp/mpt/log")
	require.NoError(t, err)
	lfb := block.NewBlock("", 2)
	lfb.ClientState = util.NewMerklePatriciaTrie(db, 1, nil)
	// set up enough nodes to exceed BatchSize
	for i := 0; i < util.BatchSize+1; i++ {
		_, err := lfb.ClientState.Insert(util.Path(fmt.Sprintf("%032d", i)), &util.SecureSerializableValue{Buffer: []byte{1}})
		require.NoError(t, err)
	}
	c := NewChainFromConfig()
	// todo: setup a real-life situation
	conf := c.ChainConfig.(*ConfigImpl)
	conf.ConfDataForTest().PruneStateBelowCount = 0
	/*
		for i := 0; i < c.BlockChain.Len(); i++ {
			c.BlockChain.Value = &block.BlockSummary{Round: i}
			c.BlockChain.Next()
		}
		c.BlockChain.Move(ch.PruneStateBelowCount)
	*/
	lfb.ClientStateHash = lfb.ClientState.GetRoot()
	c.stateDB = &failingNodeDB{db}
	c.LatestFinalizedBlock = lfb
	nodeCountBefore := 0
	db.Iterate(context.TODO(), func(ctx context.Context, key util.Key, node util.Node) error {
		nodeCountBefore++
		return nil
	})
	c.pruneClientState(node.GetNodeContext())
	nodeCountAfter := 0
	db.Iterate(context.TODO(), func(ctx context.Context, key util.Key, node util.Node) error {
		nodeCountAfter++
		return nil
	})
	assert.Equal(t, nodeCountBefore, nodeCountAfter)
}
