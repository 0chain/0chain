package ZeroChainPersistentStorage

import (
	"fmt"
	"database/sql"
	_ "github.com/lib/pq"
	"errors"
)

type ZCClient struct {
	Clientid string 
	PublicKey string 
}

func ValidateClientID (Client ZCClient) (valid bool){
	if len(Client.Clientid) > 0{
		return false
	}else{
		return true
	}
}
		
func InsertClient (NewClient ZCClient) (success bool, err error) {
	if (client_db_type == "psql") {
		return InsertClient_psql(NewClient)
	}else {
		return false, nil
	}
}

func InsertClient_psql (NewClient ZCClient) (success bool, err error) {
	if ValidateClientID(NewClient) {
		new_err := errors.New("Invalid client_id. Client_id cannot be blank.")
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
	
	_, err = db.Exec("INSERT INTO clients(public_key, client_id) VALUES($1,$2)", NewClient.PublicKey, NewClient.Clientid)

	if err != nil {
		return false, err
	}
	return true, nil
}

func GetClient (GetClient ZCClient) (ReturnedClient ZCClient, success bool, err error) {
	if (client_db_type == "psql") {
		return GetClient_psql(GetClient)
	}else {
		return ReturnedClient, false, nil
	}
}

func GetClient_psql (GetClient ZCClient) (ReturnedClient ZCClient, success bool, err error) {
	if ValidateClientID(GetClient) {
		new_err := errors.New("Invalid client_id. Client_id cannot be blank.")
		return ReturnedClient, false, new_err
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
		return ReturnedClient, false, err
	}
	//_, err = db.Exec("INSERT INTO clients(public_key, hash_key) VALUES($1,$2)", NewClient.PublicKey, NewClient.Clientid)
//	rows, err := db.Query("SELECT public_key, client_id FROM clients where client_id = " + GetClient.Clientid)
	//rows, err := db.Exec("SELECT public_key, client_id FROM clients where client_id = $1", GetClient.Clientid)

//	if len(rows) != 1 {
//		fmt.Printf("Uh oh\n");
//	}

	tx, err := db.Begin()
	if err != nil {
		return ReturnedClient, false, err
	}

	rows, err := tx.Query("SELECT public_key, client_id FROM clients where client_id = '"+GetClient.Clientid+"'")
	
	var retrieved_PublicKey string
	var retrieved_clientID string
	rows.Next()
	err = rows.Scan(&retrieved_PublicKey, &retrieved_clientID)
	if err != nil {			
		return ReturnedClient, false, err
	}
	ReturnedClient.Clientid = retrieved_clientID
	ReturnedClient.PublicKey = retrieved_PublicKey
		
	if err != nil {
		return ReturnedClient, false, err
	}
	return ReturnedClient, true, nil
}

func DelClient (DelClient ZCClient) (success bool, err error) {
	if (client_db_type == "psql") {
		return DelClient_psql(DelClient)
	}else {
		return false, nil
	}
}

func DelClient_psql (DelClient ZCClient) (success bool, err error) {
	if ValidateClientID(DelClient) {
		new_err := errors.New("Invalid client_id. Client_id cannot be blank.")
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

	stmt, err := db.Prepare("Delete from clients where client_id=$1")
	if err != nil {
		return false, err
	}

        res, del_err := stmt.Exec(DelClient.Clientid)
	if del_err != nil {
		return false, del_err
	}

	affected, affect_err := res.RowsAffected()
	if affect_err != nil {
		return false, affect_err
	}

	if affected != 1{
		new_err := errors.New("No clients Deleted.")
		return false, new_err
	}
		
	return  true, nil
}
