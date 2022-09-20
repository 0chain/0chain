package benchmark

import (
	"strconv"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

func AddMockEvents(eventDb *event.EventDb) {
	for block_number := int64(0); block_number <= viper.GetInt64(benchmark.NumBlocks); block_number++ {
		for i := int(0); i <= viper.GetInt(benchmark.NumTransactionPerBlock); i++ {
			if viper.GetBool(benchmark.EventDbEnabled) {
				event := event.Event{
					BlockNumber: block_number,
					TxHash:      GetMockTransactionHash(block_number, i),
					Type:        int(event.TypeStats),
					Tag:         3,
					Index:       "mock index",
					Data:        "mock data",
				}
				_ = eventDb.Store.Get().Create(&event)
			}
		}
	}
}

func AddMockErrors(eventDb *event.EventDb) {
	if !viper.GetBool(benchmark.EventDbEnabled) {
		return
	}
	for block_number := int64(0); block_number <= viper.GetInt64(benchmark.NumBlocks); block_number++ {
		for i := int(0); i <= viper.GetInt(benchmark.NumTransactionPerBlock); i++ {
			if viper.GetBool(benchmark.EventDbEnabled) && i%3 == 0 {
				error := event.Error{
					TransactionID: GetMockTransactionHash(block_number, i),
					Error:         "mock error text",
				}
				_ = eventDb.Store.Get().Create(&error)
			}
		}
	}
}

func AddMockTransactions(
	clients []string,
	eventDb *event.EventDb,
) {
	if !viper.GetBool(benchmark.EventDbEnabled) {
		return
	}
	const txnTypeSmartContract = 1000
	for blockNumber := int64(0); blockNumber <= viper.GetInt64(benchmark.NumBlocks); blockNumber++ {
		for i := 0; i <= viper.GetInt(benchmark.NumTransactionPerBlock); i++ {
			if viper.GetBool(benchmark.EventDbEnabled) {
				transaction := event.Transaction{
					Hash:              GetMockTransactionHash(blockNumber, i),
					BlockHash:         GetMockBlockHash(blockNumber),
					Round:             blockNumber,
					Version:           "mock version",
					ClientId:          clients[i%len(clients)],
					ToClientId:        clients[int(blockNumber)%len(clients)],
					TransactionData:   "mock transaction data",
					Signature:         "mock signature",
					CreationDate:      int64(common.Now()),
					Fee:               100,
					TransactionType:   txnTypeSmartContract,
					TransactionOutput: "mock output",
					OutputHash:        "mock output hash",
					Status:            0,
				}
				var err error
				transaction.Value, err = currency.Int64ToCoin(blockNumber)
				if err != nil {
					panic(err)
				}
				_ = eventDb.Store.Get().Create(&transaction)
			}
		}
	}
}

func AddMockBlocks(
	miners []string,
	eventDb *event.EventDb,
) {
	if !viper.GetBool(benchmark.EventDbEnabled) {
		return
	}
	for block_number := int64(0); block_number <= viper.GetInt64(benchmark.NumBlocks); block_number++ {
		if viper.GetBool(benchmark.EventDbEnabled) {
			block := event.Block{
				Hash:                  GetMockBlockHash(block_number),
				Version:               "mock version",
				CreationDate:          int64(common.Now().Duration()),
				Round:                 block_number,
				MinerID:               miners[int(block_number)%len(miners)],
				RoundRandomSeed:       block_number,
				MerkleTreeRoot:        "mock mt root",
				StateHash:             "mock state hash",
				ReceiptMerkleTreeRoot: "mock rmt root",
				NumTxns:               viper.GetInt(benchmark.NumTransactionPerBlock),
				MagicBlockHash:        "mock matic block hash",
				PrevHash:              GetMockBlockHash(block_number - 1),
				Signature:             "mock signature",
				ChainId:               "mock chain id",
				RunningTxnCount:       "mock running txn count",
				RoundTimeoutCount:     0,
				CreatedAt:             time.Now(),
			}
			_ = eventDb.Store.Get().Create(&block)
		}
	}
}

func AddMockUsers(
	clients []string,
	eventDb *event.EventDb,
) {
	if !viper.GetBool(benchmark.EventDbEnabled) {
		return
	}
	for _, client := range clients {
		user := event.User{
			UserID:  client,
			Balance: 100,
		}
		_ = eventDb.Store.Get().Create(&user)
	}
}

func GetMockBlockHash(blockNumber int64) string {
	return encryption.Hash("block" + strconv.FormatInt(blockNumber, 10))
}

func GetMockTransactionHash(blockNumber int64, index int) string {
	return encryption.Hash("block" +
		strconv.FormatInt(blockNumber, 10) + "index" + strconv.Itoa(index))
}

func GetMockWriteMarkerLookUpHash(allocationNum int, index int) string {
	return encryption.Hash("write marker look up hash" + "block" +
		strconv.Itoa(allocationNum) + "index" + strconv.Itoa(index))
}

func GetMockWriteMarkerContentHash(allocationNum int, index int) string {
	return encryption.Hash("write marker content hash" + "block" +
		strconv.Itoa(allocationNum) + "index" + strconv.Itoa(index))
}

func GetMockWriteMarkerFileName(index int) string {
	return "mock write marker file_" + strconv.Itoa(index)

}
