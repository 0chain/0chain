package worker

import (
	"0dns.io/core/state"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"0dns.io/core/config"
	. "0dns.io/core/logging"

	"go.uber.org/zap"

	"github.com/0chain/gosdk/core/block"
	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/gosdk/core/util"
)

const (
	GET_LATEST_FINALIZED_MAGIC_BLOCK = `/v1/block/get/latest_finalized_magic_block`
	GET_MAGIC_BLOCK_INFO             = `/v1/block/magic/get?`
)

func SetupWorkers(ctx context.Context) {
	go FetchMagicBlock(ctx)
}

func FetchMagicBlock(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(config.Configuration.MagicBlockWorkerTimerInSeconds) * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			magicBlock, err := GetLatestFinalizedMagicBlock(ctx)
			if err != nil {
				Logger.Error("Failed to get latest finalized magic block from blockchain", zap.Error(err))
				continue
			}

			state.SetFromCurrentMagicBlock(config.Configuration, magicBlock)
			Logger.Info("Magic block updated successfully", zap.Any("magic_block_number", magicBlock.MagicBlockNumber))
		}
	}
}

func GetLatestFinalizedMagicBlock(ctx context.Context) (m *block.MagicBlock, err error) {
	return fetchMagicBlock(ctx, GET_LATEST_FINALIZED_MAGIC_BLOCK)
}

func GetMagicBlockByNumber(ctx context.Context, number int64) (m *block.MagicBlock, err error) {
	return fetchMagicBlock(ctx, fmt.Sprintf("%smagic_block_number=%d", GET_MAGIC_BLOCK_INFO, number))
}

// fetchMagicBlock - waits till we get the magic block from all the sharders
// and returns the block which gets max consensus.
func fetchMagicBlock(ctx context.Context, query string) (m *block.MagicBlock, err error) {
	s := state.Get()

	numSharders := len(s.Sharders)
	var result = make(chan *util.GetResponse, numSharders)
	defer close(result)
	queryMagicBlockFromSharders(ctx, query, s.Sharders, result)

	var (
		maxConsensus   int
		roundConsensus = make(map[string]int)
	)

	type respObj struct {
		MagicBlock *block.MagicBlock `json:"magic_block"`
	}

	for i := 0; i < numSharders; i++ {
		var rsp = <-result

		if rsp.StatusCode != http.StatusOK {
			Logger.Error("status not ok", zap.Any("body", rsp.Body))
			continue
		}

		var respo respObj
		if err = json.Unmarshal([]byte(rsp.Body), &respo); err != nil {
			Logger.Error(" magic block parse error: ", zap.Error(err))
			err = nil
			continue
		}

		var h = encryption.FastHash([]byte(respo.MagicBlock.Hash))
		if roundConsensus[h]++; roundConsensus[h] > maxConsensus {
			maxConsensus = roundConsensus[h]
			m = respo.MagicBlock
		}
	}

	if maxConsensus == 0 {
		return nil, fmt.Errorf("magic block info not found")
	}

	return m, err
}

func queryMagicBlockFromSharders(ctx context.Context, query string, sharders []string, result chan *util.GetResponse) {
	for _, sharder := range util.Shuffle(sharders) {
		go func(sharderurl string) {
			url := fmt.Sprintf("%v%v", sharderurl, query)
			req, err := util.NewHTTPGetRequestContext(ctx, url)
			if err != nil {
				Logger.Error("failed to create new req", zap.Error(err))
				return
			}
			res, err := req.Get()
			if err != nil {
				Logger.Error("error from sharder request", zap.Error(err))
			}
			result <- res
		}(sharder)
	}
}
