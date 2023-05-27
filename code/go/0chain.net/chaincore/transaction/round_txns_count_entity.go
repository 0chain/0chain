package transaction

import (
	"context"
	"fmt"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

type RoundTxnsCount struct {
	datastore.HashIDField // Keyspaced hash of the round number - used as key
	TxnsCount  int   `json:"txns_count"`
}

const roundKeySpace = "round_txn_count"

// SetRoundKey - set the entity hash to the keyspaced hash of the round number
func BuildSummaryRoundKey(roundNumber int64) datastore.Key {
	return datastore.ToKey(
		encryption.Hash(
			fmt.Sprintf(
				"%s:%d",
				roundKeySpace,
				roundNumber,
			),
		),
	)
}

func (r *RoundTxnsCount) GetEntityMetadata() datastore.EntityMetadata {
	return transactionSummaryEntityMetadata
}

func (r *RoundTxnsCount) GetKey() datastore.Key {
	return datastore.ToKey(r.Hash)
}

func (r *RoundTxnsCount) SetKey(key datastore.Key) {
	r.Hash = datastore.ToString(key)
}

func (r *RoundTxnsCount) Read(ctx context.Context, key datastore.Key) error {
	return nil
}

func (r *RoundTxnsCount) Write(ctx context.Context) error {
	return nil
}

func (r *RoundTxnsCount) Delete(ctx context.Context) error {
	return nil
}

func (r *RoundTxnsCount) GetScore() (int64, error) {
	return 0, nil
}