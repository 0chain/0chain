package node

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/pprof"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	. "0chain.net/core/logging"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	CountErrorThresholdNodeInactive = 5
)

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *Pool) StatusMonitor(ctx context.Context) {
	np.statusMonitor(ctx)
	updateTimer := time.NewTimer(time.Second)
	monitorTimer := time.NewTimer(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-monitorTimer.C:
			np.statusMonitor(ctx)
			if np.GetActiveCount()*10 < len(np.Nodes)*8 {
				monitorTimer = time.NewTimer(5 * time.Second)
			} else {
				monitorTimer = time.NewTimer(10 * time.Second)
			}
		case <-updateTimer.C:
			np.statusUpdate(ctx)
			updateTimer = time.NewTimer(time.Second * 2)
		}
	}

}

/*OneTimeStatusMonitor - checks the status of nodes only once*/
func (np *Pool) OneTimeStatusMonitor(ctx context.Context) {
	np.statusMonitor(ctx)
}

func (np *Pool) statusUpdate(ctx context.Context) {
	np.mmx.RLock()
	nodes := np.shuffleNodes()
	np.mmx.RUnlock()
	for _, node := range nodes {
		if Self.IsEqual(node) {
			continue
		}
		if common.Within(node.GetLastActiveTime().Unix(), 10) {
			node.updateMessageTimings()
			if time.Since(node.Info.AsOf) < 60*time.Second {
				continue
			}
		}
		if node.GetErrorCount() >= CountErrorThresholdNodeInactive {
			node.SetStatus(NodeStatusInactive)
		}
	}
	np.ComputeNetworkStats()
}

func (np *Pool) statusMonitor(context.Context) {
	nodes := np.shuffleNodesLock()
	for _, node := range nodes {
		if Self.IsEqual(node) {
			continue
		}
		if common.Within(node.GetLastActiveTime().Unix(), 10) {
			node.updateMessageTimings()
			if time.Since(node.Info.AsOf) < 60*time.Second {
				continue
			}
		}
		statusURL := node.GetStatusURL()
		ts := time.Now().UTC()
		data, hash, signature, err := Self.TimeStampSignature()
		if err != nil {
			panic(err)
		}
		statusURL = fmt.Sprintf("%v?id=%v&data=%v&hash=%v&signature=%v", statusURL, Self.Underlying().GetKey(), data, hash, signature)
		resp, err := httpClient.Get(statusURL)
		if err != nil {
			node.AddErrorCount(1) // ++
			if node.IsActive() {
				if node.GetErrorCount() >= CountErrorThresholdNodeInactive {
					node.SetStatus(NodeStatusInactive)
					N2n.Error("Node inactive", zap.String("node_type", node.GetNodeTypeName()), zap.Int("set_index", node.SetIndex), zap.Any("node_id", node.GetKey()), zap.Error(err))
				}
			}
		} else {
			info := Info{}
			if err := common.FromJSON(resp.Body, &info); err == nil {
				info.AsOf = time.Now()
				node.SetInfo(info)
			}
			resp.Body.Close()
			if !node.IsActive() {
				node.SetErrorCount(0)
				node.SetStatus(NodeStatusActive)
				N2n.Info("Node active", zap.String("node_type", node.GetNodeTypeName()), zap.Int("set_index", node.SetIndex), zap.Any("key", node.GetKey()))
			}
			node.SetLastActiveTime(ts)
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
	ReadNodes(resp.Body, dnp, dnp)
	var changed = false
	for _, node := range dnp.Nodes {
		if _, ok := np.NodesMap[node.GetKey()]; !ok {
			node.SetStatus(NodeStatusActive)
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
