package node

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"0chain.net/common"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *Pool) StatusMonitor(ctx context.Context) {
	np.statusMonitor(ctx)
	ticker := time.NewTicker(10 * time.Second)
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
		MaxIdleConns:       100,
		IdleConnTimeout:    time.Minute,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Timeout: 500 * time.Millisecond}
	nodes := np.shuffleNodes()
	for _, node := range nodes {
		if node == Self.Node {
			continue
		}
		maxcount := node.GetMaxMessageCount()
		if common.Within(node.LastActiveTime.Unix(), 10) {
			var minval float32 = math.MaxFloat32
			var maxval float32
			for _, timer := range node.TimersByURI {
				if timer.Count()*10 < maxcount {
					continue
				}
				v := float32(timer.Mean())
				if v > maxval {
					maxval = v
				}
				if v < minval {
					minval = v
				}
			}
			if minval > maxval {
				minval = maxval
			}
			node.LargeMessageSendTime = maxval
			node.SmallMessageSendTime = minval
			continue
		}
		statusURL := node.GetStatusURL()
		ts := time.Now().UTC()
		data, hash, signature, err := Self.TimeStampSignature()
		if err != nil {
			panic(err)
		}
		statusURL = fmt.Sprintf("%v?id=%v&publicKey=%v&timestamp=%v&data=%v&hash=%v&signature=%v", statusURL, Self.Node.GetKey(), Self.Node.PublicKey, ts, data, hash, signature)
		resp, err := client.Get(statusURL)
		if err != nil {
			node.ErrorCount++
			if node.Status == NodeStatusActive {
				if node.ErrorCount > 5 {
					node.Status = NodeStatusInactive
					Logger.Error("Node inactive", zap.Any("node_type", node.GetNodeTypeName()), zap.Any("set_index", node.SetIndex), zap.Any("node_id", node.GetKey()), zap.Error(err))
				}
			}
		} else {
			resp.Body.Close()
			if node.Status == NodeStatusInactive {
				node.ErrorCount = 0
				node.Status = NodeStatusActive
				Logger.Info("Node active", zap.Any("node_type", node.GetNodeTypeName()), zap.Any("set_index", node.SetIndex), zap.Any("key", node.GetKey()))
			}
			node.LastActiveTime = ts
		}
	}
}

/*DownloadNodeData - downloads the node definition data for the given pool type from the given node */
func (np *Pool) DownloadNodeData(node *Node) bool {
	url := fmt.Sprintf("%v/_nh/list/%v", node.GetN2NURLBase(), node.GetNodeType())
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
			np.AddNode(node)
			changed = true
		}
	}
	if changed {
		np.ComputeProperties()
	}
	return true
}
