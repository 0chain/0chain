package wallet

import (
	"0chain.net/node"
)

func (np *Pool) RegisterAll(handler node.SendHandler) []*node.Node {
	return np.RegisterAtleast(len(np.Pool.Nodes), handler)
}

func (np *Pool) RegisterAtleast(numNodes int, handler node.SendHandler) []*node.Node {
	sentTo := make([]*node.Node, 0, numNodes)
	count := 0
	for _, node := range np.Pool.Nodes {
		if node != nil && count < numNodes {
			np.Pool.SendTo(handler, node.ID)
			sentTo = append(sentTo, node)
			count++
		}
	}
	return sentTo
}

func SendFormHandler(uri string, form map[string][]string, options *node.SendOptions) {

}
