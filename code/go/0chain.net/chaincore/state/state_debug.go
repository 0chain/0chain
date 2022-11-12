package state

import (
	"0chain.net/chaincore/config"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/util"
)

const (
	DebugLevelNone  = 0
	DebugLevelChain = iota
	DebugLevelBlock = iota
	DebugLevelTxn   = iota
	DebugLevelNode  = iota
)

var debugState = DebugLevelNone

//Debug - indicates whether state debugging is enabled or not
func Debug() bool {
	return debugState > DebugLevelNone
}

//DebugChain - indicates whether state debugging level is chain or more granular
func DebugChain() bool {
	return debugState >= DebugLevelChain
}

//DebugBlock - indicates whether state debugging level is block or more granular
func DebugBlock() bool {
	return debugState >= DebugLevelBlock
}

//DebugTxn - indicates whether state debugging level is txn or more granular
func DebugTxn() bool {
	return config.Development() && viper.GetBool("logging.verbose")
}

//DebugNode - indicates whether state debugging level is mpt node or more granular
func DebugNode() bool {
	return debugState >= DebugLevelNode
}

//SetDebugLevel - set the state debug level
func SetDebugLevel(level int) {
	debugState = level
	if DebugNode() {
		util.DebugMPTNode = true
	}
}
