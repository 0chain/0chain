package utils

import (
	"log"
	"regexp"
	"strconv"

	"0chain.net/chaincore/node"
	"0chain.net/core/encryption"

	crpc "0chain.net/conductor/conductrpc"
)

var signature = encryption.NewBLS0ChainScheme()

func init() {
	if err := signature.GenerateKeys(); err != nil {
		panic(err)
	}
}

// Sign by internal ("wrong") secret key generated randomly once client created.
func Sign(hash string) (sign string, err error) {
	return signature.Sign(hash)
}

// Split nodes list by given IsGoodOrBad.
func Split(s *crpc.State, igb crpc.IsGoodOrBad, nodes []*node.Node) (
	good, bad []*node.Node) {

	for _, n := range nodes {
		if igb.IsBad(s, n.GetKey()) {
			bad = append(bad, n)
		} else if igb.IsGood(s, n.GetKey()) {
			good = append(good, n)
		}
	}
	return
}

// Filter return IsBy nodes only.
func Filter(s *crpc.State, ib crpc.IsBy, nodes []*node.Node) (
	rest []*node.Node) {

	for _, n := range nodes {
		if ib.IsBy(s, n.GetKey()) {
			rest = append(rest, n)
		}
	}
	return
}

// IsSpammer checks whether a node is a spammer.
// A list of spammers names with format "miner-x" are passed, then the x is extracted and compared with the node index.
func IsSpammer(spammers []string, node *node.Node) (isSpammer bool) {
	isSpammer = false
	re := regexp.MustCompile("[0-9]+")

	for _, spammer := range spammers {
		indexStr := re.FindAllString(spammer, 1)
		if len(indexStr) > 0 {
			index, err := strconv.Atoi(indexStr[0])
			if err != nil {
				log.Fatalf("Spammer name should have the format miner-<1..>: %s", spammer)
				return
			}
			if index == node.SetIndex+1 {
				isSpammer = true
				return
			}
		}
	}

	return
}
