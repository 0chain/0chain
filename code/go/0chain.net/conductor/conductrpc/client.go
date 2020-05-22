package conductrpc

import (
	"fmt"

	"github.com/valyala/gorpc"
)

// Client of the conductor RPC server.
type Client struct {
	address string
	client  *gorpc.Client
	dispc   *gorpc.DispatcherClient
}

// NewClient creates new client will be interacting
// with server with given address.
func NewClient(address string) (c *Client) {
	c = new(Client)
	c.client = gorpc.NewTCPClient(address)

	var disp = gorpc.NewDispatcher()
	disp.AddFunc("onViewChange", nil)
	disp.AddFunc("onPhase", nil)
	disp.AddFunc("onAddMiner", nil)
	disp.AddFunc("onAddSharder", nil)
	disp.AddFunc("onNodeReady", nil)
	c.dispc = disp.NewFuncClient(c.client)

	return
}

// Address of RPC server.
func (c *Client) Address() string {
	return c.address
}

//
// miner SC RPC
//

func (c *Client) Phase(phase PhaseEvent) (err error) {
	_, err = c.dispc.Call("onPhase", phase)
	return
}

// ViewChange notification.
func (c *Client) ViewChange(viewChange ViewChangeEvent) (err error) {
	_, err = c.dispc.Call("onViewChange", viewChange)
	return
}

// AddMiner notification.
func (c *Client) AddMiner(add AddMinerEvent) (err error) {
	_, err = c.dispc.Call("onAddMiner", add)
	return
}

// AddSharder notification.
func (c *Client) AddSharder(add AddSharderEvent) (err error) {
	_, err = c.dispc.Call("onAddSharder", add)
	return
}

//
// nodes RPC
//

// NodeReady notification.
func (c *Client) NodeReady(nodeID NodeID) (join bool, err error) {
	var face interface{}
	if face, err = c.dispc.Call("onNodeReady", nodeID); err != nil {
		return
	}
	var ok bool
	if join, ok = face.(bool); !ok {
		return false, fmt.Errorf("invalid response type %T", face)
	}
	return
}
