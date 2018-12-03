package node

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/pprof"
	"time"

	"0chain.net/common"
	"0chain.net/logging"
	. "0chain.net/logging"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *Pool) StatusMonitor(ctx context.Context) {
	np.statusMonitor(ctx)
	timer := time.NewTimer(time.Second)
	for true {
		select {
		case <-ctx.Done():
			return
		case _ = <-timer.C:
			np.statusMonitor(ctx)
			if np.GetActiveCount()*10 < len(np.Nodes)*8 {
				timer = time.NewTimer(5 * time.Second)
			} else {
				timer = time.NewTimer(10 * time.Second)
			}
		}
	}

}

/*OneTimeStatusMonitor - checks the status of nodes only once*/
func (np *Pool) OneTimeStatusMonitor(ctx context.Context) {
	np.statusMonitor(ctx)
}

func (np *Pool) statusMonitor(ctx context.Context) {
	tr := &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    time.Minute,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Timeout: TimeoutSmallMessage}
	nodes := np.shuffleNodes()
	for _, node := range nodes {
		if node == Self.Node {
			continue
		}
		if common.Within(node.LastActiveTime.Unix(), 10) {
			node.updateMessageTimings()
			continue
		}
		statusURL := node.GetStatusURL()
		ts := time.Now().UTC()
		data, hash, signature, err := Self.TimeStampSignature()
		if err != nil {
			panic(err)
		}
		statusURL = fmt.Sprintf("%v?id=%v&data=%v&hash=%v&signature=%v", statusURL, Self.Node.GetKey(), data, hash, signature)
		resp, err := client.Get(statusURL)
		if err != nil {
			node.ErrorCount++
			if node.IsActive() {
				if node.ErrorCount > 5 {
					node.Status = NodeStatusInactive
					N2n.Error("Node inactive", zap.String("node_type", node.GetNodeTypeName()), zap.Int("set_index", node.SetIndex), zap.Any("node_id", node.GetKey()), zap.Error(err))
				}
			}
		} else {
			resp.Body.Close()
			if !node.IsActive() {
				node.ErrorCount = 0
				node.Status = NodeStatusActive
				N2n.Info("Node active", zap.String("node_type", node.GetNodeTypeName()), zap.Int("set_index", node.SetIndex), zap.Any("key", node.GetKey()))
			}
			node.LastActiveTime = ts
		}
	}
	np.ComputeNetworkStats()
}

/*DownloadNodeData - downloads the node definition data for the given pool type from the given node */
func (np *Pool) DownloadNodeData(node *Node) bool {
	url := fmt.Sprintf("%v/_nh/list/%v", node.GetN2NURLBase(), node.GetNodeType())
	client := &http.Client{Timeout: TimeoutLargeMessage}
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

func (n *Node) MemoryUsage() {
	ticker := time.NewTicker(5 * time.Minute)
	for true {
		select {
		case <-ticker.C:
			common.LogRuntime(logging.MemUsage, zap.Any(n.Description, n.SetIndex))

			// Average time duration to add go routine logs to 0chain.log file => 618.184µs
			// Average increase in file size for each update => 10 kB
			if viper.GetBool("logging.memlog") {
				buf := new(bytes.Buffer)
				pprof.Lookup("goroutine").WriteTo(buf, 1)
				logging.Logger.Info("runtime", zap.String("Go routine output", buf.String()))
			}
		}
	}
}
