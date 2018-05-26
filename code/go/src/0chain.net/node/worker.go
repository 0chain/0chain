package node

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *Pool) StatusMonitor(ctx context.Context) {
	tr := &http.Transport{
		MaxIdleConns:       1000,            // TODO: since total nodes is expected to be fixed, this may be OK
		IdleConnTimeout:    2 * time.Minute, // more than the frequency of checking will ensure always on
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Timeout: 500 * time.Millisecond}
	//ticker := time.NewTicker(2 * time.Minute)
	ticker := time.NewTicker(20 * time.Second)
	for _ = range ticker.C {
		select {
		case <-ctx.Done():
			break
		default:
		}
		nodes := np.shuffleNodes()
		activeCount := 0
		for _, node := range nodes {
			statusURL := node.GetStatusURL()
			ts := time.Now().UTC()
			data, hash, signature, err := Self.TimeStampSignature()
			if err != nil {
				panic(err)
			}
			statusURL = fmt.Sprintf("%v&data=%v&hash=%v&signature=%v", statusURL, data, hash, signature)
			_, err = client.Get(statusURL)
			if err != nil {
				node.ErrorCount++
				if node.Status == NodeStatusActive {
					if node.ErrorCount > 5 {
						node.Status = NodeStatusInactive
						fmt.Printf("error connecting to %v: %v\n", node.GetID(), err)
						fmt.Printf("node %v became inactive\n", node.GetID())
					}
				}
			} else {
				if node.Status == NodeStatusInactive {
					node.ErrorCount = 0
					node.Status = NodeStatusActive
					fmt.Printf("node %v became active\n", node.GetID())
				}
				node.LastActiveTime = ts
			}
		}

		activeCount++
		if activeCount*3 < len(nodes) {
			np.SendAtleast(1, np.DownloadNodeData)
		} else {
			//TODO: This is just to test but we do the downloading of node definitions less frequently than node status check
			np.SendAtleast(1, np.DownloadNodeData)
		}
	}
}

/*DownloadNodeData - downloads the node definition data for the given pool type from the given node */
func (np *Pool) DownloadNodeData(node *Node) bool {
	url := fmt.Sprintf("%v/_nh/list/%v", node.GetURLBase(), node.GetNodeType())
	client := &http.Client{Timeout: 2000 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	dnp := NewPool(NodeTypeMiner)
	ReadNodes(resp.Body, dnp, dnp, dnp)
	for _, node := range dnp.Nodes {
		if _, ok := np.NodesMap[node.GetID()]; !ok {
			node.Status = NodeStatusActive
			np.AddNode(node)
		}
	}
	return true
}
