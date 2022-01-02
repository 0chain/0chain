package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func GetAuthorizerNodes(state cstate.StateContextI) (*AuthorizerNodes, error) {
	authNodes := &AuthorizerNodes{}
	authNodesBytes, err := state.GetTrieNode(AllAuthorizerKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, err)
	}
	if authNodesBytes == nil {
		authNodes.NodeMap = make(map[string]*AuthorizerNode)
		return authNodes, nil
	}

	encoded := authNodesBytes.Encode()
	Logger.Info("get authorizer nodes", zap.String("hash", string(encoded)))

	err = authNodes.Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return authNodes, nil
}
