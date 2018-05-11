package ZeroChainPersistentStorage

import (
	"fmt"
	"database/sql"
	_ "github.com/lib/pq"
	"errors"
)

type ZCTransaction struct {
	Clientid string
	TransactionData string
	Createdate string
	Hash string
	Signature string
}

func ValidateHash (Transaction ZCTransaction) (valid bool){
	if len(Transaction.Hash) > 0{
		return false
	}else{
		return true
	}
}

func InsertTransaction (NewTransaction ZCTransaction) (success bool, err error) {
	if (transaction_db_type == "psql") {
		return InsertTransaction_psql(NewTransaction)
	}else {
		return false, nil
	}
}

func InsertTransaction_psql (NewTransaction ZCTransaction) (success bool, err error) {
	if ValidateHash(NewTransaction){
		new_err := errors.New("Invalid hash. Hash cannot be blank.")
		return false, new_err
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", 
		psql_host, 
		psql_port, 
		psql_user, 
		psql_password, 
		psql_dbname)
	
	db, err := sql.Open("postgres", psqlInfo)

	defer db.Close()
	if err != nil {
		return false, err
	}
	
	_, err = db.Exec("INSERT INTO transactions(client_id, transaction_data, createdate, hash, signature) VALUES($1,$2,$3,$4,$5)", 
		NewTransaction.Clientid, 
		NewTransaction.TransactionData,
		NewTransaction.Createdate,
		NewTransaction.Hash,
		NewTransaction.Signature)
	
	if err != nil {
		return false, err
	}
	return true, nil
}


func GetTransaction (GetTransaction ZCTransaction) (ReturnedTransaction ZCTransaction, success bool, err error) {
	if (transaction_db_type == "psql") {
		return GetTransaction_psql(GetTransaction)
	}else {
		return ReturnedTransaction, false, nil
	}
}

func GetTransaction_psql (GetTransaction ZCTransaction) (ReturnedTransaction ZCTransaction, success bool, err error) {
	if ValidateHash(GetTransaction){
		new_err := errors.New("Invalid hash. Hash cannot be blank.")
		return ReturnedTransaction, false, new_err
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable connect_timeout=60", 
		psql_host, 
		psql_port, 
		psql_user, 
		psql_password, 
		psql_dbname)

	db, err := sql.Open("postgres", psqlInfo)
	defer db.Close()
	if err != nil {
		return ReturnedTransaction, false, err
	}

	tx, err := db.Begin()
	if err != nil {
		return ReturnedTransaction, false, err
	}

	rows, err := tx.Query("SELECT client_id, transaction_data, createdate, hash, signature  FROM transactions where hash = '"+GetTransaction.Hash+"'")
	
	var client_id string
	var transaction_data string
	var createdate string
	var hash string
	var signature string
	rows.Next()
	err = rows.Scan(&client_id, &transaction_data, &createdate, &hash, &signature)
	if err != nil {			
		return ReturnedTransaction, false, err
	}
	ReturnedTransaction.Clientid = client_id
	ReturnedTransaction.TransactionData = transaction_data
	ReturnedTransaction.Createdate = createdate
	ReturnedTransaction.Hash = hash
	ReturnedTransaction.Signature = signature
		
	if err != nil {
		return ReturnedTransaction, false, err
	}
	return ReturnedTransaction, true, nil
}

func DelTransaction (DelTransaction ZCTransaction) (success bool, err error) {
	if (transaction_db_type == "psql") {
		return DelTransaction_psql(DelTransaction)
	}else {
		return false, nil
	}
}

func DelTransaction_psql (DelTransaction ZCTransaction) (success bool, err error) {
	if ValidateHash(DelTransaction){
		new_err := errors.New("Invalid hash. Hash cannot be blank.")
		return false, new_err
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable connect_timeout=60", 
		psql_host, 
		psql_port, 
		psql_user, 
		psql_password, 
		psql_dbname)

	db, err := sql.Open("postgres", psqlInfo)
	defer db.Close()
	if err != nil {
		return false, err
	}

	stmt, err := db.Prepare("Delete from transactions where hash=$1")
	if err != nil {
		return false, err
	}

        res, del_err := stmt.Exec(DelTransaction.Hash)
	if del_err != nil {
		return false, del_err
	}

	affected, affect_err := res.RowsAffected()
	if affect_err != nil {
		return false, affect_err
	}

	if affected != 1{
		new_err := errors.New("No Transactions Deleted.")
		return false, new_err
	}
		
	return  true, nil
}

