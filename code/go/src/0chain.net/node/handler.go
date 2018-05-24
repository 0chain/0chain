package node

import (
	"net/http"
	"strings"

	"0chain.net/common"
)

func SetupHandlers() {
	http.HandleFunc("/_nh/status", StatusHandler)
	http.HandleFunc("/_nh/whoami", WhoAmIHandler)
	http.HandleFunc("/_nh/list/m", GetMinersHandler)
	http.HandleFunc("/_nh/list/s", GetShardersHandler)
	http.HandleFunc("/_nh/list/b", GetBlobbersHandler)
}

/*StatusHandler - allows checking the status of the node */
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		return
	}
	publicKey := r.FormValue("publicKey")
	timestamp := r.FormValue("timestamp")
	ts := common.Now()
	ts.Parse([]byte(timestamp))
	data := r.FormValue("data")
	hash := r.FormValue("hash")
	signature := r.FormValue("signature")
	addressParts := strings.Split(r.RemoteAddr, ":")
	node := Miners.GetNode(id)
	if node == nil {
		node = Sharders.GetNode(id)
		if node == nil {
			node = Blobbers.GetNode(id)
		}
	}
	if node == nil {
		// TODO: This doesn't allow adding new nodes that weren't already known.
		return
	}
	if node.Host != addressParts[0] {
		// TODO: Node's ip address changed. Should we update ourselves?
	}
	if node.PublicKey == publicKey {
		ok, err := node.Verify(ts, data, hash, signature)
		if !ok || err != nil {
			return
		}
		node.LastActiveTime = common.Now()
	} else {
		// TODO: private/public keys changed by the node. Should we update ourselves?
	}
}

//WhoAmIHandler - who am i?
func WhoAmIHandler(w http.ResponseWriter, r *http.Request) {
	if Self == nil {
		return
	}
	Self.Print(w)
}

/*GetMinersHandler - get the list of known miners */
func GetMinersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	Miners.Print(w)
}

/*GetShardersHandler - get the list of known sharders */
func GetShardersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	Sharders.Print(w)
}

/*GetBlobbersHandler - get the list of known blobbers */
func GetBlobbersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	Blobbers.Print(w)
}
