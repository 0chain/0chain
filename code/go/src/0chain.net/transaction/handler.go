package transaction

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"go.uber.org/zap"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/transaction/get", common.UserRateLimit(common.ToJSONResponse(memorystore.WithConnectionHandler(GetTransaction))))
}

/*GetTransaction - given an id returns the transaction information */
func GetTransaction(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, transactionEntityMetadata, "hash")
}

/*PutTransaction - Given a transaction data, it stores it */
func PutTransaction(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	txn, ok := entity.(*Transaction)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", entity)
	}
	txn.ComputeProperties()
	debugTxn := txn.DebugTxn()
	err := txn.Validate(ctx)

	if err != nil {
		if debugTxn {
			Logger.Error("put transaction (debug transaction)", zap.String("txn", txn.Hash), zap.Error(err))
		}
		return nil, err
	}
	if debugTxn {
		Logger.Info("put transaction (debug transaction)", zap.String("txn", txn.Hash), zap.String("txn_obj", datastore.ToJSON(txn).String()))
	}
	if datastore.DoAsync(ctx, txn) {
		TransactionCount++
		return txn, nil
	}
	err = entity.GetEntityMetadata().GetStore().Write(ctx, txn)
	if err != nil {
		return nil, err
	}
	TransactionCount++
	return txn, nil
}
