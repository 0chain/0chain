package conductrpc

import (
	"net/rpc"

	"0chain.net/conductor/conductrpc/stats"
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

// addAuthorizer notification.
func (c *client) addAuthorizer(add *AddAuthorizerEvent) (err error) {
	err = c.client.Call("Server.AddAuthorizer", add, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddAuthorizer", add, &struct{}{})
	}
	return
}

// sharderKeep notification
func (c *client) sharderKeep(sk *SharderKeepEvent) (err error) {
	err = c.client.Call("Server.SharderKeep", sk, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.SharderKeep", sk, &struct{}{})
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

func (c *client) configureTestCase(blob []byte) (err error) {
	err = c.client.Call("Server.ConfigureTestCase", blob, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.ConfigureTestCase", blob, &struct{}{})
	}
	return
}

func (c *client) addTestCaseResult(blob []byte) (err error) {
	err = c.client.Call("Server.AddTestCaseResult", blob, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddTestCaseResult", blob, &struct{}{})
	}
	return
}

func (c *client) addBlockServerStats(ss *stats.BlockRequest) (err error) {
	err = c.client.Call("Server.AddBlockServerStats", ss, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddBlockServerStats", ss, &struct{}{})
	}
	return
}

func (c *client) addVRFSServerStats(ss *stats.VRFSRequest) (err error) {
	err = c.client.Call("Server.AddVRFSServerStats", ss, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddVRFSServerStats", ss, &struct{}{})
	}
	return
}

func (c *client) addBlockClientStats(req []byte) (err error) {
	err = c.client.Call("Server.AddBlockClientStats", req, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddBlockClientStats", req, &struct{}{})
	}
	return
}

func (c *client) magicBlock() (configFile *string, err error) {
	err = c.client.Call("Server.MagicBlock", &struct{}{}, &configFile)
	for err == rpc.ErrShutdown {
		err = c.client.Call("Server.MagicBlock", &struct{}{}, &configFile)
	}
	return
}
