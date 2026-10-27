package node

import (
	"strings"
	"sync"
)

// Host rewrites let a network move its nodes to a new domain without a new magic block.
// A magic block's hash covers node IDs, keys and shares but not Host/N2NHost (see
// MagicBlock.GetHashBytes), so a node may substitute hostnames locally and still agree on
// every block. Configured with `network.host_rewrite` in 0chain.yaml as a list of
// "<from-suffix>=<to-suffix>" pairs, e.g. ".0chain.net=.zus.network". Empty = no-op.
var (
	hostRewriteMu sync.RWMutex
	hostRewrites  [][2]string
)

// SetHostRewrites installs the host suffix rewrites. Malformed entries are skipped.
func SetHostRewrites(pairs []string) {
	rw := make([][2]string, 0, len(pairs))
	for _, p := range pairs {
		from, to, ok := strings.Cut(strings.TrimSpace(p), "=")
		from, to = strings.ToLower(strings.TrimSpace(from)), strings.ToLower(strings.TrimSpace(to))
		if !ok || from == "" || to == "" {
			continue
		}
		rw = append(rw, [2]string{from, to})
	}
	hostRewriteMu.Lock()
	hostRewrites = rw
	hostRewriteMu.Unlock()
}

// RewriteHost returns host with the first matching suffix rewrite applied.
func RewriteHost(host string) string {
	hostRewriteMu.RLock()
	defer hostRewriteMu.RUnlock()
	if len(hostRewrites) == 0 {
		return host
	}
	lh := strings.ToLower(host)
	for _, rw := range hostRewrites {
		if strings.HasSuffix(lh, rw[0]) {
			return host[:len(host)-len(rw[0])] + rw[1]
		}
	}
	return host
}

// applyHostRewrite rewrites the node's own Host and N2NHost in place.
func (n *Node) applyHostRewrite() {
	n.Host = RewriteHost(n.Host)
	n.N2NHost = RewriteHost(n.N2NHost)
}
