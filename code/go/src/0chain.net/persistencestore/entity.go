package persistencestore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"0chain.net/datastore"
)

/*PersistenceEntity - Persistence Entity */
type PersistenceEntity interface {
	datastore.Entity
	PRead(ctx context.Context, key datastore.Key) error
	PWrite(ctx context.Context) error
	PDelete(ctx context.Context) error
}

func PRead(ctx context.Context, key datastore.Key, entity PersistenceEntity) error {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	err := json.NewDecoder(buffer).Decode(entity)
	c := GetCon(ctx)
	var errs []string
	fmt.Println("Printing Entity contents here  : ", entity)

	if err != nil {
		panic(err)
	}
	if err := c.Query(
		"SELECT JSON * from block").Exec(); err != nil {
		errs = append(errs, err.Error())
	}
	return nil
}

func PWrite(ctx context.Context, entity PersistenceEntity) error {
	return PWriteAux(ctx, entity, true)
}

func PWriteAux(ctx context.Context, entity PersistenceEntity, overwrite bool) error {
	buffer := datastore.ToJSON(entity)
	c := GetCon(ctx)
	var errs []string

	fmt.Println("PWrite here : ", buffer)

	jData, err := json.Marshal(entity)
	if err != nil {
		panic(err)
	}
	//fmt.Println("PWrite here jData: ", jData)

	if err := c.Query(
		"INSERT INTO block JSON ?", jData).Exec(); err != nil {
		errs = append(errs, err.Error())
	} else {
		overwrite = true
	}
	fmt.Println("errors : ", errs)
	return nil
}

func PDelete(ctx context.Context, entity PersistenceEntity) error {
	c := GetCon(ctx)
	c.Query("Delete from block")
	return nil
}
