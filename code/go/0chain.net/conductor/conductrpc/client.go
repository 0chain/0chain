package conductrpc

import (
	"net/rpc"
)

// client of the conductor RPC server.
type client struct {
	address string      // RPC server address
	client  *rpc.Client // RPC client
}

// newClient creates new client will be interacting
// with server with given address.
func newClient(address string) (c *client, err error) {
	if address, err = Host(address); err != nil {
		return
	}
	c = new(client)
	if c.client, err = rpc.Dial("tcp", address); err != nil {
		return nil, err
	}
	c.address = address
	return
}

func (c *client) dial() (err error) {
	c.client, err = rpc.Dial("tcp", c.address)
	return
}

// Address of RPC server.
func (c *client) Address() string {
	return c.address
}

//
// miner SC RPC
//

// phase change event
func (c *client) phase(phase *PhaseEvent) (err error) {
	err = c.client.Call("Server.Phase", phase, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.Phase", phase, &struct{}{})
	}
	return
}

// viewChange notification.
func (c *client) viewChange(viewChange *ViewChangeEvent) (err error) {
	err = c.client.Call("Server.ViewChange", viewChange, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.ViewChange", viewChange, &struct{}{})
	}
	return
}

// addMiner notification.
func (c *client) addMiner(add *AddMinerEvent) (err error) {
	err = c.client.Call("Server.AddMiner", add, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddMiner", add, &struct{}{})
	}
	return
}

// addSharder notification.
func (c *client) addSharder(add *AddSharderEvent) (err error) {
	err = c.client.Call("Server.AddSharder", add, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddSharder", add, &struct{}{})
	}
	return
}

// addBlobber notification.
func (c *client) addBlobber(add *AddBlobberEvent) (err error) {
	err = c.client.Call("Server.AddBlobber", add, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddBlobber", add, &struct{}{})
	}
	return
}

// round notification.
func (c *client) round(re *RoundEvent) (err error) {
	err = c.client.Call("Server.Round", re, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.Round", re, &struct{}{})
	}
	return
}

// contributeMPK phase notification.
func (c *client) contributeMPK(cmpke *ContributeMPKEvent) (err error) {
	err = c.client.Call("Server.ContributeMPK", cmpke, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.ContributeMPK", cmpke, &struct{}{})
	}
	return
}

// shareOrSignsShares phase notification.
func (c *client) shareOrSignsShares(sosse *ShareOrSignsSharesEvent) (err error) {
	err = c.client.Call("Server.ShareOrSignsShares", sosse, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.ShareOrSignsShares", sosse, &struct{}{})
	}
	return
}

// state requests current client state using long polling strategy. E.g.
// when the state had updated, then the method returns.
func (c *client) state(me NodeID) (state *State, err error) {
	err = c.client.Call("Server.State", me, &state)
	for err == rpc.ErrShutdown {
		err = c.client.Call("Server.State", me, &state)
	}
	return
}
