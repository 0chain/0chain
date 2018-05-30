package persistencestore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
)

var providers = make(map[string]common.EntityProvider)

/*RegisterEntityProvider - keep track of a list of entity providers. An entity can be registered with multiple names
* as long as two entities don't use the same name
 */
func RegisterEntityProvider(entityName string, provider common.EntityProvider) {
	providers[entityName] = provider
}

/*GetProvider - return the provider registered for the given entity */
func GetProvider(entityName string) common.EntityProvider {
	return providers[entityName]
}

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

	fmt.Println("here : ", entity)

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
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	json.NewEncoder(buffer).Encode(entity)
	c := GetCon(ctx)
	var errs []string

	fmt.Println("here : ", entity)

	b, err := json.Marshal(entity)
	if err != nil {
		panic(err)
	}

	if err := c.Query(
		"INSERT INTO block JSON ?", b).Exec(); err != nil {
		errs = append(errs, err.Error())
	} else {
		overwrite = true
	}
	return nil
}

func PDelete(ctx context.Context, entity PersistenceEntity) error {
	c := GetCon(ctx)
	c.Query("Delete from block")
	return nil
}
