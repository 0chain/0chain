package node

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

//SetupHandlers - setup all the handlers
func SetupHandlers() {
	http.HandleFunc("/_nh/whoami", common.UserRateLimit(WhoAmIHandler))
	http.HandleFunc("/_nh/status", common.UserRateLimit(StatusHandler))
	http.HandleFunc("/_nh/getpoolmembers", common.UserRateLimit(common.ToJSONResponse(GetPoolMembersHandler)))
}

//WhoAmIHandler - who am i?
func WhoAmIHandler(w http.ResponseWriter, r *http.Request) {
	if Self == nil {
		return
	}
	Self.Print(w)
}

func scale(val int64) float64 {
	return float64(val) / 1000000.0
}

//PrintSendStats - print the n2n statistics to this node
func (n *Node) PrintSendStats(w io.Writer) {
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
		sizer := n.GetSizeMetric(uri)
		if sizer != nil {
			fmt.Fprintf(w, "<td class='number'>%d</td>", sizer.Min())
			fmt.Fprintf(w, "<td class='number'>%.2f &plusmn;%.2f</td>", sizer.Mean(), sizer.StdDev())
			fmt.Fprintf(w, "<td class='number'>%d</td>", sizer.Max())
		}
		fmt.Fprintf(w, "</tr>")
	}
}

/*StatusHandler - allows checking the status of the node */
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		return
	}
	nd := GetNode(id)
	if nd == nil {
		return
	}
	if nd.IsActive() {
		common.Respond(w, r, Self.Node.Info, nil)
		return
	}
	data := r.FormValue("data")
	hash := r.FormValue("hash")
	signature := r.FormValue("signature")
	if data == "" || hash == "" || signature == "" {
		return
	}
	if ok, _ := Self.ValidateSignatureTime(data); !ok {
		return
	}
	/*
		addressParts := strings.Split(r.RemoteAddr, ":")
		if nd.Host != addressParts[0] {
			return
		} */
	if ok, err := nd.Verify(signature, hash); !ok || err != nil {
		return
	}
	nd.LastActiveTime = time.Now().UTC()
	if nd.Status == NodeStatusInactive {
		nd.Status = NodeStatusActive
		N2n.Info("Node active", zap.String("node_type", nd.GetNodeTypeName()), zap.Int("set_index", nd.SetIndex), zap.Any("key", nd.GetKey()))
	}
	common.Respond(w, r, Self.Node.Info, nil)
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
