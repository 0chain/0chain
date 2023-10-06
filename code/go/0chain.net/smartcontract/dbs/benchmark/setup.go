package benchmark

import (
	"strconv"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

func AddMockEvents(eventDb *event.EventDb) {
	if !viper.GetBool(benchmark.EventDbEnabled) {
		return
	}

	var events []event.Event
	for round := benchmark.GetOldestAggregateRound(); round < viper.GetInt64(benchmark.NumBlocks); round++ {
		for i := 0; i <= viper.GetInt(benchmark.NumTransactionPerBlock); i++ {
			events = append(events, event.Event{
				BlockNumber: round,
				TxHash:      GetMockTransactionHash(round, i),
				Type:        event.TypeStats,
				Tag:         3,
				Index:       "mock index",
				Data:        "mock data",
			})

		}
	}
	if res := eventDb.Store.Get().Create(&events); res.Error != nil {
		log.Fatal("adding mock events", res.Error)
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
	const txnTxnSmartContract = 1000
	for blockNumber := int64(1); blockNumber <= viper.GetInt64(benchmark.NumBlocks); blockNumber++ {
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
					Nonce:             int64(i),
					TransactionType:   txnTxnSmartContract,
					TransactionOutput: "mock output",
					OutputHash:        "mock output hash",
					Status:            0,
				}
				var err error
				transaction.Value, err = currency.Int64ToCoin(blockNumber)
				if err != nil {
					panic(err)
				}
				err = eventDb.Store.Get().Create(&transaction).Error
				if err != nil {
					panic(err)
				}
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
	for blockNumber := int64(1); blockNumber <= viper.GetInt64(benchmark.NumBlocks); blockNumber++ {
		if viper.GetBool(benchmark.EventDbEnabled) {
			block := event.Block{
				Hash:                  GetMockBlockHash(blockNumber),
				Version:               "mock version",
				CreationDate:          int64(common.Now().Duration()),
				Round:                 blockNumber,
				MinerID:               miners[int(blockNumber)%len(miners)],
				RoundRandomSeed:       blockNumber,
				MerkleTreeRoot:        "mock mt root",
				StateHash:             "mock state hash",
				ReceiptMerkleTreeRoot: "mock rmt root",
				NumTxns:               viper.GetInt(benchmark.NumTransactionPerBlock),
				MagicBlockHash:        "mock matic block hash",
				PrevHash:              GetMockBlockHash(blockNumber - 1),
				Signature:             "mock signature",
				ChainId:               "mock chain id",
				StateChangesCount:     33,
				RunningTxnCount:       "mock running txn count",
				RoundTimeoutCount:     0,
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
	var users []event.User
	for _, client := range clients {
		user := event.User{
			UserID:  client,
			Balance: 100,
		}
		users = append(users, user)
	}
	if res := eventDb.Store.Get().Create(&users); res.Error != nil {
		log.Fatal(res.Error)
	}
	andMockUserSnapshots(users, eventDb)
}

func andMockUserSnapshots(users []event.User, edb *event.EventDb) {
	if edb == nil {
		return
	}
	var aggregates []event.UserAggregate
	for _, user := range users {
		aggregate := event.UserAggregate{
			Round:  viper.GetInt64(benchmark.NumBlocks) - 1,
			UserID: user.UserID,
		}
		aggregates = append(aggregates, aggregate)
	}

	res := edb.Store.Get().Create(&aggregates)
	if res.Error != nil {
		log.Fatal(res.Error)
	}
}

func GetMockBlockHash(blockNumber int64) string {
	return encryption.Hash("block" + strconv.FormatInt(blockNumber, 10))
}

func GetMockTransactionHash(blockNumber int64, index int) string {
	return encryption.Hash("block" +
		strconv.FormatInt(blockNumber, 10) + "index" + strconv.Itoa(index))
}

func AddAggregatePartitions(edb *event.EventDb) {
	var (
		period      = viper.GetInt(benchmark.EventDbPartitionChangePeriod)
		keep        = viper.GetInt(benchmark.EventDbPartitionKeepCount)
		blocks      = viper.GetInt64(benchmark.NumBlocks)
		firstPeriod = benchmark.GetOldestAggregateRound()
	)

	for i := 0; i < keep; i++ {
		round := firstPeriod + int64(i*period)
		if round < 0 {
			continue
		} else if round > blocks {
			break
		}

		edb.AddPartitions(round)
	}
}
