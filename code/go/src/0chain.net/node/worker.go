package node

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "0chain.net/logging"
	"go.uber.org/zap"
)

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *Pool) StatusMonitor(ctx context.Context) {
	//ticker := time.NewTicker(2 * time.Minute)
	ticker := time.NewTicker(20 * time.Second)
	for true {
		select {
		case <-ctx.Done():
			return
		case _ = <-ticker.C:
			np.statusMonitor(ctx)
		}
	}
}

func (np *Pool) statusMonitor(ctx context.Context) {
	tr := &http.Transport{
		MaxIdleConns:       1000,            // TODO: since total nodes is expected to be fixed, this may be OK
		IdleConnTimeout:    2 * time.Minute, // more than the frequency of checking will ensure always on
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Timeout: 500 * time.Millisecond}
	nodes := np.shuffleNodes()
	activeCount := 0
	for _, node := range nodes {
		if node == Self.Node {
			continue
		}
		statusURL := node.GetStatusURL()
		ts := time.Now().UTC()
		data, hash, signature, err := Self.TimeStampSignature()
		if err != nil {
			panic(err)
		}
		statusURL = fmt.Sprintf("%v&data=%v&hash=%v&signature=%v", statusURL, data, hash, signature)
		resp, err := client.Get(statusURL)
		if err != nil {
			node.ErrorCount++
			if node.Status == NodeStatusActive {
				if node.ErrorCount > 5 {
					node.Status = NodeStatusInactive

					Logger.Error("error connecting", zap.Any("to node", node.GetNodeTypeName()), zap.Any("Node index", node.SetIndex), zap.Any("key", node.GetKey()), zap.Error(err))
					//fmt.Printf("error connecting to %v node(%v): %v %v\n", node.GetNodeTypeName(), node.SetIndex, node.GetKey(), err)
					Logger.Error("Node inactive", zap.Any("to node", node.GetNodeTypeName()), zap.Any("Node index", node.SetIndex), zap.Any("key", node.GetKey()))
					//fmt.Printf("%v node(%v) %v became inactive\n", node.GetNodeTypeName(), node.SetIndex, node.GetKey())
				}
			}
		} else {
			resp.Body.Close()
			if node.Status == NodeStatusInactive {
				node.ErrorCount = 0
				node.Status = NodeStatusActive
				Logger.Info("Node active", zap.Any("Node", node.GetNodeTypeName()), zap.Any("Node index", node.SetIndex), zap.Any("key", node.GetKey()))
				//fmt.Printf("%v node(%v) %v became active\n", node.GetNodeTypeName(), node.SetIndex, node.GetKey())
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

/*DownloadNodeData - downloads the node definition data for the given pool type from the given node */
func (np *Pool) DownloadNodeData(node *Node) bool {
	url := fmt.Sprintf("%v/_nh/list/%v", node.GetURLBase(), node.GetNodeType())
	client := &http.Client{Timeout: 2000 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	dnp := NewPool(NodeTypeMiner)
	ReadNodes(resp.Body, dnp, dnp, dnp)
	var changed = false
	for _, node := range dnp.Nodes {
		if _, ok := np.NodesMap[node.GetKey()]; !ok {
			node.Status = NodeStatusActive
			//fmt.Printf("Discovered a new node:%v , %v \n", node.GetKey(), node.SetIndex)
			np.AddNode(node)
			changed = true
		}
	}
	if changed {
		np.ComputeProperties()
	}
	return true
}
