package node

import "testing"

func TestRewriteHost(t *testing.T) {
	SetHostRewrites(nil)
	if got := RewriteHost("miner4.0chain.net"); got != "miner4.0chain.net" {
		t.Fatalf("no rewrites configured: got %q", got)
	}

	SetHostRewrites([]string{" .0chain.net = .zus.network ", "bad-entry", "=.x", ".foo.io="})
	defer SetHostRewrites(nil)
	cases := map[string]string{
		"miner4.0chain.net":      "miner4.zus.network",
		"SHARDER1.0CHAIN.NET":    "SHARDER1.zus.network",
		"mb01.0chainstaking.net": "mb01.0chainstaking.net", // suffix must match ".0chain.net", not "0chain.net"
		"miner1.bytepatch.io":    "miner1.bytepatch.io",
		"66.51.159.41":           "66.51.159.41",
		"":                       "",
	}
	for in, want := range cases {
		if got := RewriteHost(in); got != want {
			t.Errorf("RewriteHost(%q) = %q, want %q", in, got, want)
		}
	}

	n := &Node{Host: "miner4.0chain.net", N2NHost: "miner4.0chain.net", Port: 7071}
	n.applyHostRewrite()
	if n.Host != "miner4.zus.network" || n.N2NHost != "miner4.zus.network" {
		t.Fatalf("applyHostRewrite: got %q / %q", n.Host, n.N2NHost)
	}
	if got := (&Node{N2NHost: "miner9.0chain.net", Port: 7071}).GetN2NURLBase(); got != "http://miner9.zus.network:7071" {
		t.Fatalf("GetN2NURLBase = %q", got)
	}
}
