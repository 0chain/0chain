package miner

import "0chain.net/chaincore/transaction"

const (
	payFeesTxnName               = "payFees"
	commitSettingsChangesTxnName = "commit_settings_changes"
	blobberBlockRewardsTxnName   = "blobber_block_rewards"
	generateChallengeTxnName     = "generate_challenge"
)

var gBuildInTxnsMap = map[string]struct{}{
	payFeesTxnName:               {},
	commitSettingsChangesTxnName: {},
	blobberBlockRewardsTxnName:   {},
	generateChallengeTxnName:     {},
}

// isBuildInTxn checks if the txn is build-in txn.
func (mc *Chain) isBuildInTxn(txn *transaction.Transaction) bool {
	if txn.TransactionType != transaction.TxnTypeSmartContract {
		return false
	}

	_, ok := gBuildInTxnsMap[txn.FunctionName]
	return ok
}
