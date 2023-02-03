package node

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/pprof"
	"time"

	"go.uber.org/zap"

	"0chain.net/core/common"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
)

const (
	CountErrorThresholdNodeInactive = 5
)

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *Pool) StatusMonitor(ctx context.Context, startRound int64, waitC chan struct{}) {
	logging.N2n.Debug("[monitor] start status monitor",
		zap.Int64("starting round", startRound),
		zap.String("node type", NodeTypeNames[np.Type].Code))
	np.statusMonitor(ctx, startRound)
	updateTimer := time.NewTimer(time.Second)
	monitorTimer := time.NewTimer(time.Second)
	for {
		select {
		case <-ctx.Done():
			logging.N2n.Debug("[monitor] status monitor canceled, StatusMonitor",
				zap.String("node type", NodeTypeNames[np.Type].Code),
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
	np.mmx.Lock()
	select {
	case <-ctx.Done():
		np.mmx.Unlock()
		return
	default:
		for _, node := range np.Nodes {
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
	}
	np.mmx.Unlock()
	np.ComputeNetworkStats()
}

func (np *Pool) statusMonitor(ctx context.Context, startRound int64) {
	logging.N2n.Debug("[monitor] status monitor for", zap.Int64("starting round", startRound))
	nodes := np.shuffleNodes(true)
	for i, node := range nodes {
		select {
		case <-ctx.Done():
			logging.N2n.Debug("[monitor] status monitor canceled - statusMonitor",
				zap.String("node type", NodeTypeNames[np.Type].Code),
				zap.Int64("starting round", startRound))
			return
		default:
		}

		if Self.IsEqual(node) {
			continue
		}
		if common.Within(node.GetLastActiveTime().Unix(), 10) {
			nodes[i].updateMessageTimings()
			if time.Since(node.Info.AsOf) < 60*time.Second {
				logging.N2n.Debug("node active check - active",
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
			logging.N2n.Error("node active check - failed to create request",
				zap.Int64("start round", startRound),
				zap.String("node host", node.Host),
				zap.Int("node port", node.Port),
				zap.String("node n2n host", node.N2NHost))
			continue
		}

		func(nd *Node) {
			reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			req = req.WithContext(reqCtx)
			resp, err := httpClient.Do(req)
			if err != nil {
				nd.AddErrorCount(1) // ++
				var nodeInActive bool
				if nd.GetErrorCount() >= CountErrorThresholdNodeInactive {
					nd.SetStatus(NodeStatusInactive)
					nodeInActive = true
				}

				logging.N2n.Debug("node active check - ping failed",
					zap.Error(err),
					zap.Bool("node is inactive", nodeInActive),
					zap.Int64("start round", startRound),
					zap.String("node_type", nd.GetNodeTypeName()),
					zap.String("node host", nd.Host),
					zap.Int("node port", nd.Port),
					zap.String("node n2n host", nd.N2NHost),
					zap.Int64("ErrCount", nd.GetErrorCount()),
					zap.Int64("ErrThresholdCount", CountErrorThresholdNodeInactive),
					zap.String("pool miners pointer", fmt.Sprintf("%p", np)))
				return
			}
			defer resp.Body.Close()

			logging.N2n.Debug("node active check - ping success",
				zap.Int64("start round", startRound),
				zap.String("node host", nd.Host),
				zap.Int("node port", nd.Port),
				zap.String("node n2n host", nd.N2NHost),
				zap.String("miners  pointer", fmt.Sprintf("%p", np)))
			info := Info{}
			if err := common.FromJSON(resp.Body, &info); err == nil {
				info.AsOf = time.Now()
				nd.SetInfo(info)
			}
			if !nd.IsActive() {
				logging.N2n.Info("Node active",
					zap.String("node_type", nd.GetNodeTypeName()),
					zap.Int("set_index", nd.SetIndex),
					zap.String("key", nd.GetKey()))
			}
			nd.SetErrorCount(0)
			nd.SetStatus(NodeStatusActive)
			nd.SetLastActiveTime(ts)
		}(nodes[i])
	}
	np.ComputeNetworkStats()
}

func (n *Node) MemoryUsage() {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		<-ticker.C
		common.LogRuntime(logging.MemUsage, zap.Int(n.Description, n.SetIndex))

		// Average time duration to add go routine logs to 0chain.log file => 618.184µs
		// Average increase in file size for each update => 10 kB
		if viper.GetBool("logging.memlog") {
			buf := new(bytes.Buffer)
			_ = pprof.Lookup("goroutine").WriteTo(buf, 1)
			logging.Logger.Info("runtime", zap.String("Go routine output", buf.String()))
		}
	}
}
