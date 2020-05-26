package conductrpc

import (
	"net/rpc"
)

// Client of the conductor RPC server.
type Client struct {
	address string
	client  *rpc.Client
}

// NewClient creates new client will be interacting
// with server with given address.
func NewClient(address string) (c *Client, err error) {
	if address, err = Host(address); err != nil {
		return
	}
	c = new(Client)
	if c.client, err = rpc.Dial("tcp", address); err != nil {
		return nil, err
	}
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
	return c.client.Call("Phase", phase, &struct{}{})
}

// ViewChange notification.
func (c *Client) ViewChange(viewChange *ViewChangeEvent) (err error) {
	return c.client.Call("ViewChange", viewChange, &struct{}{})
}

// AddMiner notification.
func (c *Client) AddMiner(add *AddMinerEvent) (err error) {
	return c.client.Call("AddMiner", add, &struct{}{})
}

// AddSharder notification.
func (c *Client) AddSharder(add *AddSharderEvent) (err error) {
	return c.client.Call("AddSharder", add, &struct{}{})
}

// NodeReady notification.
func (c *Client) NodeReady(nodeID NodeID) (join bool, err error) {
	err = c.client.Call("NodeReady", nodeID, &join)
	return
}
