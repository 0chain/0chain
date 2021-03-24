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
func (np *Pool) StatusMonitor(ctx context.Context, startRound int64, waitC chan struct{}) {
	N2n.Debug("[monitor] start status monitor",
		zap.Int64("starting round", startRound),
		zap.Any("node type", NodeTypeNames[np.Type].Code))
	np.statusMonitor(ctx, startRound)
	updateTimer := time.NewTimer(time.Second)
	monitorTimer := time.NewTimer(time.Second)
	for {
		select {
		case <-ctx.Done():
			N2n.Debug("[monitor] status monitor canceled, StatusMonitor",
				zap.Any("node type", NodeTypeNames[np.Type].Code),
				zap.Int64("start round", startRound))
			close(waitC)
			return
		case <-monitorTimer.C:
			np.statusMonitor(ctx, startRound)
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
func (np *Pool) OneTimeStatusMonitor(ctx context.Context, startRound int64) {
	np.statusMonitor(ctx, startRound)
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

func (np *Pool) statusMonitor(ctx context.Context, startRound int64) {
	N2n.Debug("[monitor] status monitor for", zap.Int64("starting round", startRound))
	nodes := np.shuffleNodesLock()
	for _, node := range nodes {
		select {
		case <-ctx.Done():
			N2n.Debug("[monitor] status monitor canceled - statusMonitor",
				zap.Any("node type", NodeTypeNames[np.Type].Code),
				zap.Int64("starting round", startRound))
			return
		default:
		}

		if Self.IsEqual(node) {
			continue
		}
		if common.Within(node.GetLastActiveTime().Unix(), 10) {
			node.updateMessageTimings()
			if time.Since(node.Info.AsOf) < 60*time.Second {
				N2n.Debug("node active check - active",
					zap.Int64("start round", startRound),
					zap.String("node host", node.Host),
					zap.Int("node port", node.Port),
					zap.String("node n2n host", node.N2NHost))
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
		req, err := http.NewRequest(http.MethodGet, statusURL, nil)
		if err != nil {
			N2n.Error("node active check - failed to create request",
				zap.Int64("start round", startRound),
				zap.String("node host", node.Host),
				zap.Int("node port", node.Port),
				zap.String("node n2n host", node.N2NHost))
			continue
		}
		reqctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		req = req.WithContext(reqctx)
		errC := make(chan error)
		respC := make(chan *http.Response)
		go func() {
			resp, err := httpClient.Do(req)
			if err != nil {
				errC <- err
				return
			}

			respC <- resp
		}()

		select {
		case resp := <-respC:
			N2n.Debug("node active check - ping success",
				zap.Int64("start round", startRound),
				zap.String("node host", node.Host),
				zap.Int("node port", node.Port),
				zap.String("node n2n host", node.N2NHost))
			info := Info{}
			if err := common.FromJSON(resp.Body, &info); err == nil {
				info.AsOf = time.Now()
				node.SetInfo(info)
			}
			resp.Body.Close()
			if !node.IsActive() {
				N2n.Info("Node active", zap.String("node_type", node.GetNodeTypeName()), zap.Int("set_index", node.SetIndex), zap.Any("key", node.GetKey()))
			}
			node.SetErrorCount(0)
			node.SetStatus(NodeStatusActive)
			node.SetLastActiveTime(ts)
		case err = <-errC:
			node.AddErrorCount(1) // ++
			N2n.Debug("node active check - ping failed",
				zap.Int64("start round", startRound),
				zap.String("node host", node.Host),
				zap.Int("node port", node.Port),
				zap.String("node n2n host", node.N2NHost),
				zap.Int64("ErrCount", node.GetErrorCount()),
				zap.Int64("ErrThresholdCount", CountErrorThresholdNodeInactive),
				zap.Error(err))
			if node.GetErrorCount() >= CountErrorThresholdNodeInactive {
				node.SetStatus(NodeStatusInactive)
				N2n.Error("node active check - node inactive!!",
					zap.Int64("start round", startRound),
					zap.String("node_type", node.GetNodeTypeName()),
					zap.String("node host", node.Host),
					zap.Int("node port", node.Port),
					zap.String("node n2n host", node.N2NHost),
					zap.Int("set_index", node.SetIndex),
					zap.Any("node_id", node.GetKey()),
					zap.Int64("err_count", node.GetErrorCount()),
					zap.Error(err))
			}
		case <-time.NewTimer(5 * time.Second).C:
			cancel()
			node.AddErrorCount(1) // ++
			N2n.Debug("node active check - ping failed, context timeout",
				zap.Int64("start round", startRound),
				zap.String("node host", node.Host),
				zap.Int("node port", node.Port),
				zap.String("node n2n host", node.N2NHost),
				zap.Int64("ErrCount", node.GetErrorCount()),
				zap.Int64("ErrThresholdCount", CountErrorThresholdNodeInactive))

			if node.GetErrorCount() >= CountErrorThresholdNodeInactive {
				node.SetStatus(NodeStatusInactive)
				N2n.Error("node active check - node inactive!!, context timeout",
					zap.Int64("start round", startRound),
					zap.String("node_type", node.GetNodeTypeName()),
					zap.String("node host", node.Host),
					zap.Int("node port", node.Port),
					zap.String("node n2n host", node.N2NHost),
					zap.Int("set_index", node.SetIndex),
					zap.Any("node_id", node.GetKey()))
			}
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
