package smartcontract

import (
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"0chain.net/chaincore/node"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
)

var blsPublicKeys []string

func init() {
	logging.InitLogging("testing", "")
	blsPublicKeys = make([]string, 10)
	for i := 0; i < 10; i++ {
		ss := encryption.NewBLS0ChainScheme()
		ss.GenerateKeys()
		blsPublicKeys[i] = ss.GetPublicKey()
	}
}

func makeTestNode(typ node.NodeType, pk string) (*node.Node, error) {
	nc := map[interface{}]interface{}{
		"type":        typ,
		"public_ip":   "public ip",
		"n2n_ip":      "n2n_ip",
		"port":        8080,
		"id":          "",
		"public_key":  pk,
		"description": "description",
	}
	n, err := node.NewNode(nc)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func TestBCContext_GetNodepoolInfo(t *testing.T) {
	t.Parallel()

	mn, err := makeTestNode(node.NodeTypeMiner, blsPublicKeys[0])
	if err != nil {
		t.Fatal(err)
	}
	node.RegisterNode(mn)

	sn, err := makeTestNode(node.NodeTypeSharder, blsPublicKeys[1])
	if err != nil {
		t.Fatal(err)
	}
	node.RegisterNode(sn)

	bn, err := makeTestNode(node.NodeTypeBlobber, blsPublicKeys[2])
	if err != nil {
		t.Fatal(err)
	}
	node.RegisterNode(bn)

	makeTestMembers := func() *PoolMembersInfo {
		members := &PoolMembersInfo{
			MembersInfo: []PoolMemberInfo{
				{
					N2NHost:   bn.N2NHost,
					Port:      strconv.Itoa(bn.Port),
					Type:      "Blobber",
					PublicKey: bn.PublicKey,
				},
				{
					N2NHost:   mn.N2NHost,
					Port:      strconv.Itoa(mn.Port),
					Type:      "Miner",
					PublicKey: mn.PublicKey,
				},
				{
					N2NHost:   sn.N2NHost,
					Port:      strconv.Itoa(sn.Port),
					Type:      "Sharder",
					PublicKey: sn.PublicKey,
				},
			},
		}

		return members
	}

	tests := []struct {
		name string
		want interface{}
	}{
		{
			name: "OK",
			want: makeTestMembers(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bc := &BCContext{}
			gotI := bc.GetNodepoolInfo()
			got, ok := gotI.(*PoolMembersInfo)
			if !ok {
				t.Fatal("expected *PoolMembersInfo type")
			}
			want, ok := tt.want.(*PoolMembersInfo)
			if !ok {
				t.Fatal("expected *PoolMembersInfo type")
			}

			sort.Slice(got.MembersInfo,
				func(i, j int) bool {
					return strings.Compare(got.MembersInfo[i].Type, got.MembersInfo[j].Type) > 0
				},
			)
			sort.Slice(want.MembersInfo,
				func(i, j int) bool {
					return strings.Compare(want.MembersInfo[i].Type, want.MembersInfo[j].Type) > 0
				},
			)

			assert.Equal(t, tt.want, got)
		})
	}
}
