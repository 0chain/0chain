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
	disp.AddFunc("onViewChange", func(*ViewChangeEvent) {})
	disp.AddFunc("onPhase", func(*PhaseEvent) {})
	disp.AddFunc("onAddMiner", func(*AddMinerEvent) {})
	disp.AddFunc("onAddSharder", func(*AddSharderEvent) {})
	disp.AddFunc("onNodeReady", func(NodeID) (join bool) { return })
	c.dispc = disp.NewFuncClient(c.client)
	c.address = address

	return
}

// Address of RPC server.
func (c *Client) Address() string {
	return c.address
}

//
// miner SC RPC
//

func (c *Client) Phase(phase *PhaseEvent) (err error) {
	_, err = c.dispc.Call("onPhase", phase)
	return
}

// ViewChange notification.
func (c *Client) ViewChange(viewChange *ViewChangeEvent) (err error) {
	_, err = c.dispc.Call("onViewChange", viewChange)
	return
}

// AddMiner notification.
func (c *Client) AddMiner(add *AddMinerEvent) (err error) {
	_, err = c.dispc.Call("onAddMiner", add)
	return
}

// AddSharder notification.
func (c *Client) AddSharder(add *AddSharderEvent) (err error) {
	_, err = c.dispc.Call("onAddSharder", add)
	return
}

//
// nodes RPC
//

// NodeReady notification.
func (c *Client) NodeReady(nodeID NodeID) (join bool, err error) {
	var face interface{}
	println("FUCK IT'S HERE!", c.address)
	if face, err = c.dispc.Call("onNodeReady", nodeID); err != nil {
		return
	}
	println("SHIT IT IS NOT!")
	var ok bool
	if join, ok = face.(bool); !ok {
		return false, fmt.Errorf("invalid response type %T", face)
	}
	return
}
