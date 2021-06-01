package node

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"0chain.net/miner/minerGRPC"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	"go.uber.org/zap"
)

//SetupHandlers - setup all the handlers
func SetupHandlers() {
	svc := newGRPCMinerNodeService()
	http.HandleFunc("/_nh/whoami", common.UserRateLimit(WhoAmIHandler(svc)))
	http.HandleFunc("/_nh/status", common.UserRateLimit(StatusHandler))
	http.HandleFunc("/_nh/getpoolmembers", common.UserRateLimit(common.ToJSONResponse(GetPoolMembersHandler)))
}

//WhoAmIHandler - who am i?
func WhoAmIHandler(svc *minerNodeGRPCService) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.WhoAmI(context.Background(), &minerGRPC.WhoAmIRequest{})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Error - " + err.Error()))
			return
		}

		if resp.Data != "" {
			w.WriteHeader(http.StatusOK)
			reader := strings.NewReader(resp.Data)
			_, _ = io.Copy(w, reader)
		}
	}
}

func scale(val int64) float64 {
	return float64(val) / 1000000.0
}

//PrintSendStats - print the n2n statistics to this node
func (n *Node) PrintSendStats(w io.Writer) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	uris := make([]string, 0, len(n.TimersByURI))
	for uri := range n.TimersByURI {
		uris = append(uris, uri)
	}
	sort.SliceStable(uris, func(i, j int) bool { return uris[i] < uris[j] })
	for _, uri := range uris {
		timer := n.TimersByURI[uri]
		if timer.Count() == 0 {
			continue
		}
		fmt.Fprintf(w, "<tr>")
		fmt.Fprintf(w, "<td>%v</td>", uri)
		fmt.Fprintf(w, "<td class='number'>%9d</td>", timer.Count())
		fmt.Fprintf(w, "<td class='number'>%.2f</td>", scale(timer.Min()))
		fmt.Fprintf(w, "<td class='number'>%.2f &plusmn;%.2f</td>", timer.Mean()/1000000., timer.StdDev()/1000000.)
		fmt.Fprintf(w, "<td class='number'>%.2f</td>", scale(timer.Max()))
		sizer := n.getSizeMetric(uri)
		if sizer != nil {
			fmt.Fprintf(w, "<td class='number'>%d</td>", sizer.Min())
			fmt.Fprintf(w, "<td class='number'>%.2f &plusmn;%.2f</td>", sizer.Mean(), sizer.StdDev())
			fmt.Fprintf(w, "<td class='number'>%d</td>", sizer.Max())
		}
		fmt.Fprintf(w, "</tr>")
	}
}

// func respondWithTimeout(tm time.Duration, respond func()) {
// 	var (
// 		done  = make(chan struct{})
// 		timer = time.NewTimer(tm)
// 	)
// 	defer timer.Stop()
// 	go func() {
// 		defer close(done)
// 		respond()
// 	}()
// 	select {
// 	case <-done:
// 	case <-timer.C:
// 	}
// }

/*StatusHandler - allows checking the status of the node */
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		logging.N2n.Error("status handler -- missing id", zap.Any("from", r.RemoteAddr))
		return
	}
	nd := GetNode(id)
	if nd == nil {
		logging.N2n.Error("status handler -- node nil", zap.Any("id", id))
		return
	}
	if nd.IsActive() {
		info := Self.Underlying().Info
		logging.N2n.Info("status handler -- sending data", zap.Any("data", info))
		common.Respond(w, r, info, nil)
		return
	}
	data := r.FormValue("data")
	hash := r.FormValue("hash")
	signature := r.FormValue("signature")
	if data == "" || hash == "" || signature == "" {
		logging.N2n.Error("status handler -- missing fields", zap.Any("data", data), zap.Any("hash", hash), zap.Any("signature", signature), zap.String("node_type", nd.GetNodeTypeName()), zap.Int("set_index", nd.SetIndex), zap.Any("key", nd.GetKey()))
		return
	}
	if ok, err := ValidateSignatureTime(data); !ok {
		logging.N2n.Error("status handler -- validate time failed", zap.Any("error", err), zap.String("node_type", nd.GetNodeTypeName()), zap.Int("set_index", nd.SetIndex), zap.Any("key", nd.GetKey()))
		return
	}
	/*
		addressParts := strings.Split(r.RemoteAddr, ":")
		if nd.Host != addressParts[0] {
			return
		} */
	if ok, err := nd.Verify(signature, hash); !ok || err != nil {
		logging.N2n.Error("status handler -- signature failed", zap.Any("error", err), zap.String("node_type", nd.GetNodeTypeName()), zap.Int("set_index", nd.SetIndex), zap.Any("key", nd.GetKey()))
		return
	}
	nd.SetLastActiveTime(time.Now().UTC())
	if nd.GetStatus() == NodeStatusInactive {
		nd.SetStatus(NodeStatusActive)
		logging.N2n.Info("Node active", zap.String("node_type", nd.GetNodeTypeName()), zap.Int("set_index", nd.SetIndex), zap.Any("key", nd.GetKey()))
	}
	info := Self.Underlying().Info
	logging.N2n.Info("status handler -- sending data", zap.Any("data", info))
	common.Respond(w, r, info, nil)
}

//ToDo: Move this to MagicBlock logic
// PoolMembers of pool
type PoolMembers struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

//GetPoolMembersHandler API to get access information of all the members of the pool.
func GetPoolMembersHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	pm := &PoolMembers{}

	for _, n := range nodes {
		if n.Type == NodeTypeMiner {
			pm.Miners = append(pm.Miners, n.GetN2NURLBase())
		} else if n.Type == NodeTypeSharder {
			pm.Sharders = append(pm.Sharders, n.GetN2NURLBase())
		}
	}
	//Logger.Info("returning number of ", zap.Int("miners", len(pm.Miners)), zap.Int("node", Self.SetIndex))
	return pm, nil
}
