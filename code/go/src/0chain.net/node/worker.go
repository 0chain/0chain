package node

import (
	"fmt"
	"net/http"
	"time"

	"0chain.net/common"
)

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *Pool) StatusMonitor() {
	tr := &http.Transport{
		MaxIdleConns:       1000,            // TODO: since total nodes is expected to be fixed, this may be OK
		IdleConnTimeout:    2 * time.Minute, // more than the frequency of checking will ensure always on
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Timeout: 500 * time.Millisecond}
	//ticker := time.NewTicker(2 * time.Minute)
	ticker := time.NewTicker(20 * time.Second)
	for _ = range ticker.C {
		nodes := np.shuffleNodes()
		activeCount := 0
		for _, node := range nodes {
			statusURL := node.GetStatusURL()
			data, hash, signature, err := Self.TimeStampSignature(Self.privateKey)
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
				node.LastActiveTime = common.Now()
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
	ReadNodes(resp.Body, np, np, np)
	return true
}

/*Miners - this is the pool of miners */
var Miners = NewPool(NodeTypeMiner)

/*Sharders - this is the pool of sharders */
var Sharders = NewPool(NodeTypeSharder)

/*Blobbers - this is the pool of blobbers */
var Blobbers = NewPool(NodeTypeBlobber)
