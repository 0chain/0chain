package transaction

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/transaction/get", common.ToJSONResponse(datastore.WithConnectionHandler(GetTransaction)))
	http.HandleFunc("/v1/transaction/put", common.ToJSONEntityReqResponse(datastore.DoAsyncEntityJSONHandler(datastore.WithConnectionEntityJSONHandler(PutTransaction), TransactionEntityChannel), Provider))
}

/*SetupSharderHandlers sets up the necessary API end points for Sharders */
func SetupSharderHandlers() {
	http.HandleFunc("/v1/transaction/search", common.ToJSONResponse(datastore.WithConnectionHandler(GetTransactions)))
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
	if !common.Within(int64(txn.CreationDate), TXN_TIME_TOLERANCE) {
		return nil, common.InvalidRequest("Transaction creation time not within tolerance")
	}
	err := txn.Validate(ctx)
	if err != nil {
		return nil, err
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

/*GetTransactions - returns a list of transactions for a client
*	//TODO: This is currently implemented via the miner's datastore cache
*	//I think this should be handled by sharders who have access to historic data.
*	//Also, the sharder data is stored in NoSQL with index on both txn.ClientID and txn.ToClientID
*	//So, it should be a query of the form
*	// select * from transactions where (client_id == ? or to_client_id == ?) order by creation_date desc;
*	// This handler might support other filtering capabilities such as txns between start_date and end_date;
*	// Date based filtering is required for scalability
 */
func GetTransactions(ctx context.Context, r *http.Request) (interface{}, error) {
	client_id := r.FormValue("client_id")
	client_id_key := datastore.ToKey(client_id)
	txns := make([]*Transaction, 0, 1)
	var txnIterHandler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		select {
		case <-ctx.Done():
			datastore.GetCon(ctx).Close()
			return false
		default:
		}
		txn, ok := qe.(*Transaction)
		if !ok {
			return true
		}
		if datastore.IsEqual(txn.ClientID, client_id_key) || datastore.IsEqual(txn.ToClientID, client_id_key) {
			txns = append(txns, txn)
			if len(txns) > 5 {
				return false
			}
		}
		return true
	}
	txn := Provider().(*Transaction)
	txn.ChainID = datastore.ToKey(config.GetServerChainID())
	collectionName := txn.GetCollectionName()
	//TODO: 10 seconds is a lot but OK for testing.
	//But because this is off of redis and we don't have good filtering capability, we have to settle for large time.
	ctx, cancelf := context.WithTimeout(ctx, 10*time.Second)
	defer cancelf()
	err := datastore.IterateCollection(ctx, collectionName, txnIterHandler, Provider)
	if err != nil {
		return nil, err
	}
	return txns, nil
}
