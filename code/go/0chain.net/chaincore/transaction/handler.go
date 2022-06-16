package transaction

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"go.uber.org/zap"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/transaction", common.UserRateLimit(common.ToJSONResponse(memorystore.WithConnectionHandler(GetTransaction))))
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

	if err := txn.ComputeProperties(); err != nil {
		logging.Logger.Error("put transaction error", zap.String("txn", txn.Hash), zap.Error(err))
		return nil, err
	}

	debugTxn := txn.DebugTxn()
	err := txn.Validate(ctx)
	if err != nil {
		logging.Logger.Error("put transaction error", zap.String("txn", txn.Hash), zap.Error(err))
		return nil, err
	}
	if debugTxn {
		logging.Logger.Info("put transaction (debug transaction)", zap.String("txn", txn.Hash), zap.String("txn_obj", datastore.ToJSON(txn).String()))
	}

	cli, err := txn.GetClient(ctx)
	if err != nil || cli == nil || cli.PublicKey == "" {
		return nil, common.NewError("put transaction error", fmt.Sprintf("client %v doesn't exist, please register", txn.ClientID))
	}
	if datastore.DoAsync(ctx, txn) {
		IncTransactionCount()
		return txn, nil
	}
	err = entity.GetEntityMetadata().GetStore().Write(ctx, txn)
	if err != nil {
		logging.Logger.Info("put transaction", zap.Any("error", err), zap.Any("txn", txn.Hash), zap.Any("txn_obj", datastore.ToJSON(txn).String()))
		return nil, err
	}

	IncTransactionCount()
	return txn, nil
}

func PutTransactionWithoutVerifySig(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	txn, ok := entity.(*Transaction)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", entity)
	}

	if err := txn.ComputeProperties(); err != nil {
		logging.Logger.Error("put transaction error", zap.Error(err))
		return nil, err
	}

	debugTxn := txn.DebugTxn()
	if debugTxn {
		logging.Logger.Info("put transaction (debug transaction)", zap.String("txn", txn.Hash), zap.String("txn_obj", datastore.ToJSON(txn).String()))
	}
	cli, err := txn.GetClient(ctx)
	if err != nil || cli == nil || cli.PublicKey == "" {
		return nil, common.NewError("put transaction error", fmt.Sprintf("client %v doesn't exist, please register", txn.ClientID))
	}

	if datastore.DoAsync(ctx, txn) {
		IncTransactionCount()
		return txn, nil
	}
	err = entity.GetEntityMetadata().GetStore().Write(ctx, txn)
	if err != nil {
		logging.Logger.Info("put transaction", zap.Any("error", err), zap.Any("txn", txn.Hash), zap.Any("txn_obj", datastore.ToJSON(txn).String()))
		return nil, err
	}

	IncTransactionCount()
	return txn, nil
}
