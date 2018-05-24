package node

/*SendHandler is used to send any message to a given node */
type SendHandler func(n *Node) bool

/*SendAtleast - It tries to communicate to at least the given number of active nodes */
func (np *Pool) SendAtleast(numNodes int, handler SendHandler) {
	const THRESHOLD = 2
	nodes := np.shuffleNodes()
	validCount := 0
	allCount := 0
	for _, node := range nodes {
		if node.Status == NodeStatusInactive {
			continue
		}
		allCount++
		valid := handler(node)
		if valid {
			validCount++
			if validCount == numNodes {
				break
			}
		}
		if allCount >= numNodes+THRESHOLD {
			break
		}
	}
}
