package node

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

//SetupHandlers - setup all the handlers
func SetupHandlers() {
	http.HandleFunc("/_nh/whoami", WhoAmIHandler)
	http.HandleFunc("/_nh/status", StatusHandler)
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

//PrintSendStats - print the send statistics to this node
func (n *Node) PrintSendStats(w io.Writer) {
	for uri, timer := range n.TimersByURI {
		fmt.Fprintf(w, "<tr>")
		fmt.Fprintf(w, "<td>%v</td>", uri)
		fmt.Fprintf(w, "<td class='number'>%9d</td>", timer.Count())
		fmt.Fprintf(w, "<td class='number'>%.2f</td>", scale(timer.Min()))
		fmt.Fprintf(w, "<td class='number'>%.2f &plusmn;%.2f</td>", timer.Mean()/1000000., timer.StdDev()/1000000.)
		fmt.Fprintf(w, "<td class='number'>%.2f</td>", scale(timer.Max()))
		fmt.Fprintf(w, "</tr>")
	}
}

/*StatusHandler - allows checking the status of the node */
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		return
	}
	publicKey := r.FormValue("publicKey")
	data := r.FormValue("data")
	hash := r.FormValue("hash")
	signature := r.FormValue("signature")
	if data == "" || hash == "" || signature == "" {
		return
	}
	if ok, _ := Self.ValidateSignatureTime(data); !ok {
		return
	}
	nd := GetNode(id)
	if nd == nil {
		return
	}
	/*
		addressParts := strings.Split(r.RemoteAddr, ":")
		if nd.Host != addressParts[0] {
			return
		} */
	if nd.PublicKey != publicKey {
		return
	}
	ok, err := nd.Verify(signature, hash)
	if !ok || err != nil {
		return
	}
	nd.LastActiveTime = time.Now().UTC()
	if nd.Status == NodeStatusInactive {
		nd.Status = NodeStatusActive
	}
}
