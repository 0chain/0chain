package transaction

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/transaction/get", common.ToJSONResponse(datastore.WithConnectionHandler(GetTransaction)))
	http.HandleFunc("/v1/transaction/put", common.ToJSONEntityReqResponse(datastore.DoAsyncEntityJSONHandler(datastore.WithConnectionEntityJSONHandler(PutTransaction), TransactionEntityChannel), Provider))
}

/*GetTransaction - given an id returns the transaction information */
func GetTransaction(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, Provider, "hash")
}

/*TXN_TIME_TOLERANCE - the txn creation date should be within 5 seconds before/after of current time */
const TXN_TIME_TOLERANCE = 5

/*PutTransaction - Given a transaction data, it stores it */
func PutTransaction(ctx context.Context, object interface{}) (interface{}, error) {
	txn, ok := object.(*Transaction)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", object)
	}
	txn.ComputeProperties()
	err := txn.Validate(ctx)
	if err != nil {
		return nil, err
	}
	deltaTime := int64(time.Since(txn.CreationDate.Time) / time.Second)
	if deltaTime < -TXN_TIME_TOLERANCE || deltaTime > TXN_TIME_TOLERANCE {
		return nil, common.InvalidRequest("Transaction creation time not within tolerance")
	}
	if datastore.DoAsync(ctx, txn) {
		return txn, nil
	}
	err = datastore.Write(ctx, txn)
	if err != nil {
		return nil, err
	}
	return txn, nil
}
