package minersc

import (
	"encoding/hex"
	"testing"

	"0chain.net/smartcontract/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPooler struct {
	ids map[string]struct{}
}

func (mp *mockPooler) HasNode(id string) bool {
	_, ok := mp.ids[id]
	return ok
}

func newMockPooler(ids []string) *mockPooler {
	mp := &mockPooler{
		ids: make(map[string]struct{}),
	}

	for _, id := range ids {
		mp.ids[id] = struct{}{}
	}

	return mp
}

func createTestSimpleNodesAndNodePool() (SimpleNodes, Pooler) {
	sn := NewSimpleNodes()
	sn["0"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "0",
		},
		TotalStaked: 12,
	}
	sn["1"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "1",
		},
		TotalStaked: 10,
	}
	sn["2"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "2",
		},
		TotalStaked: 8,
	}
	sn["3"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "3",
		},
		TotalStaked: 5,
	}
	sn["4"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "4",
		},
		TotalStaked: 3,
	}
	sn["5"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "5",
		},
		TotalStaked: 3,
	}
	sn["6"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "6",
		},
		TotalStaked: 2,
	}
	sn["7"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "7",
		},
		TotalStaked: 2,
	}
	sn["8"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "8",
		},
		TotalStaked: 2,
	}
	sn["9"] = &SimpleNode{
		Provider: &provider.Provider{
			ID: "9",
		},
		TotalStaked: 1,
	}

	np := newMockPooler([]string{"6", "9", "4", "2"})

	return sn, np
}

func TestSimpleNodesReduce(t *testing.T) {
	var pmbrss int64 = 123456789

	// select up to 5 of the existing nodes
	sn, np := createTestSimpleNodesAndNodePool()
	sn.reduce(7, 0.7, pmbrss, np)
	for _, n := range sn {
		assert.Contains(t, []string{"2", "4", "6", "9", "0", "1", "3"}, n.ID)
	}

	// select up to 3 nodes from previous set and rest by desc stake
	sn, np = createTestSimpleNodesAndNodePool()
	sn.reduce(5, 0.6, pmbrss, np)
	for _, n := range sn {
		assert.Contains(t, []string{"2", "4", "6", "0", "1"}, n.ID)
	}

	// select up to 5 nodes from previous set and rest by desc stake
	sn, np = createTestSimpleNodesAndNodePool()
	sn.reduce(8, 0.6, pmbrss, np)
	for _, n := range sn {
		assert.Contains(t, []string{"2", "4", "6", "9", "0", "1", "3", "5"}, n.ID)
	}

	// select up to 6 nodes form previous set (4), and rest by desc stake
	// resolve equal stake (7:2, 8:2) using pmbrss
	sn, np = createTestSimpleNodesAndNodePool()
	sn.reduce(9, 0.6, pmbrss, np)
	for _, n := range sn {
		assert.Contains(t, []string{"2", "4", "6", "9", "0", "1", "3", "5", "8"}, n.ID)
	}

	// select up to 6 nodes form previous set (4), and rest by desc stake
	// resolve equal stake (7:2, 8:2) using pmbrss+2
	sn, np = createTestSimpleNodesAndNodePool()
	sn.reduce(9, 0.6, pmbrss+2, np)
	for _, n := range sn {
		assert.Contains(t, []string{"2", "4", "6", "9", "0", "1", "3", "5", "7"}, n.ID)
	}

}

func TestQuickFixDuplicateHosts(t *testing.T) {
	node := func(id, n2nhost, host string, port int) *MinerNode {
		return &MinerNode{SimpleNode: &SimpleNode{
			Provider: &provider.Provider{
				ID: id,
			},
			N2NHost: n2nhost,
			Host:    host,
			Port:    port,
		}}
	}
	nodes := func() []*MinerNode {
		return []*MinerNode{
			{
				SimpleNode: &SimpleNode{
					Provider: &provider.Provider{},
					N2NHost:  "abc.com",
					Host:     "lmn.com",
					Port:     0,
				},
			},
		}
	}
	assert.EqualError(t, quickFixDuplicateHosts(node("", "", "", 0), nodes()), "invalid n2nhost: ''")
	assert.EqualError(t, quickFixDuplicateHosts(node("", "localhost", "", 0), nodes()), "invalid n2nhost: 'localhost'")
	assert.EqualError(t, quickFixDuplicateHosts(node("", "127.0.0.1", "", 0), nodes()), "invalid n2nhost: '127.0.0.1'")
	assert.NoError(t, quickFixDuplicateHosts(node("", "xyz.com", "", 0), nodes()))
	assert.NoError(t, quickFixDuplicateHosts(node("", "xyz.com", "localhost", 0), nodes()))
	assert.NoError(t, quickFixDuplicateHosts(node("", "xyz.com", "127.0.0.1", 0), nodes()))
	assert.NoError(t, quickFixDuplicateHosts(node("", "xyz.com", "prq.com", 0), nodes()))
	assert.EqualError(t, quickFixDuplicateHosts(node("abc", "abc.com", "", 0), nodes()), "n2nhost:port already exists: 'abc.com:0'")
	assert.NoError(t, quickFixDuplicateHosts(node("", "abc.com", "", 1), nodes()))
	assert.EqualError(t, quickFixDuplicateHosts(node("mn", "lmn.com", "", 0), nodes()), "host:port already exists: 'lmn.com:0'")
	assert.NoError(t, quickFixDuplicateHosts(node("", "lmn.com", "", 1), nodes()))

}

func TestValidateSimpleNode(t *testing.T) {
	sn := &SimpleNode{
		Provider: &provider.Provider{
			ID: "",
		},
	}
	assert.Error(t, sn.Validate(), "id is empty")
	sn = &SimpleNode{
		Provider: &provider.Provider{
			ID: "66dfd72",
		},
	}
	assert.Error(t, sn.Validate(), "len(id) < 64")

	sn = &SimpleNode{
		Provider: &provider.Provider{
			ID: "g6dfd726644496052930658c565e02b1528a0eff832b991fdab4fd265034b214",
		},
	}
	assert.Error(t, sn.Validate(), "invalid hexadecimal")
	sn = &SimpleNode{
		Provider: &provider.Provider{
			ID: "66dfd726644496052930658c565e02b1528a0eff832b991fdab4fd265034b214",
		},
	}
	assert.NoError(t, sn.Validate(), "len(id) == 64")
}

func TestMinerNodeEncode(t *testing.T) {
	var data = "7b2273696d706c655f6d696e6572223a7b2270726f7669646572223a7b226964223a22222c2270726f76696465725f74797065223a307d2c226e326e5f686f7374223a22746573746e657432332e6465766e65742d30636861696e2e6e6574222c22686f7374223a22746573746e657432332e6465766e65742d30636861696e2e6e6574222c22706f7274223a33313230322c2267656f6c6f636174696f6e223a7b226c61746974756465223a32382e363434382c226c6f6e676974756465223a37372e3231363732317d2c2270617468223a226d696e65723032222c227075626c69635f6b6579223a223663373564343639633832383332353438613939623465373438363662663132393036373463623930353438333638656665636237313663376261363836306532653636396634653365373365303062303061343337643632623833346531643461316430383339393736613539316335306333333539656663373139373230222c2273686f72745f6e616d65223a22746573746e6574323340676d61696c2e636f6d222c226275696c645f746167223a2261363265303866623338663933653138316665663938656436323936656432393963383538383734222c22746f74616c5f7374616b65223a302c2264656c657465223a66616c73652c226e6f64655f74797065223a226d696e6572222c226c6173745f6865616c74685f636865636b223a313634313036353737382c226c6173745f73657474696e675f7570646174655f726f756e64223a307d2c227374616b655f706f6f6c223a7b22706f6f6c73223a7b7d2c2272657761726473223a302c2273657474696e6773223a7b2264656c65676174655f77616c6c6574223a22222c226d696e5f7374616b65223a302c226d61785f7374616b65223a302c226e756d5f64656c656761746573223a302c22736572766963655f636861726765223a307d2c226d696e746572223a307d7d"
	v, err := hex.DecodeString(data)
	require.NoError(t, err)

	mn := NewMinerNode()
	err = mn.Decode(v)
	require.NoError(t, err)
	data2 := mn.Encode()

	require.Equal(t, v, data2)
}
