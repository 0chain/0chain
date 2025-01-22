package transaction

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"github.com/0chain/common/core/logging"
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

func GetTransactionByHash(ctx context.Context, hash string) (interface{}, error) {
	// check if txn is invalid future txn
	if IsInvalidFutureTxn(hash) {
		return nil, errors.New("invalid future transaction")
	}

	tem := datastore.GetEntityMetadata("txn")
	if tem == nil {
		return nil, nil
	}

	cctx := memorystore.WithConnection(ctx)
	defer memorystore.Close(cctx)
	return datastore.GetEntityByHash(cctx, tem, hash)
}

/*PutTransaction - Given a transaction data, it stores it */
func PutTransaction(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	txn, ok := entity.(*Transaction)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", entity)
	}

	if txn.DebugTxn() {
		logging.Logger.Info("put transaction", zap.Any("txn", txn))
	} else {
		logging.Logger.Info("put transaction", zap.String("txn", txn.Hash))
	}

	if datastore.DoAsync(ctx, txn) {
		IncTransactionCount()
		return txn, nil
	}

	err := entity.GetEntityMetadata().GetStore().Write(ctx, txn)
	if err != nil {
		logging.Logger.Error("put transaction", zap.Error(err), zap.String("txn", txn.Hash), zap.String("txn_obj", datastore.ToJSON(txn).String()))
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
		logging.Logger.Error("put transaction", zap.Error(err), zap.String("txn", txn.Hash), zap.String("txn_obj", datastore.ToJSON(txn).String()))
		return nil, err
	}

	IncTransactionCount()
	return txn, nil
}
