package persistencestore

import (
	"context"
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

type PersistenceEntity interface {
	datastore.Entity
	Read(ctx context.Context, key datastore.Key) error
	Write(ctx context.Context) error
	Delete(ctx context.Context) error
}

func GetEntityKey(entity PersistenceEntity) datastore.Key {
	var key interface{} = entity.GetKey()
	switch v := key.(type) {
	case string:
		return datastore.ToKey(fmt.Sprintf("%s:%v", entity.GetEntityName(), v))
	case []byte:
		return datastore.ToKey(append(append([]byte(entity.GetEntityName()), ':'), v...))
	default:
		return datastore.EmptyKey
	}
}

func Write(w http.ResponseWriter, r *http.Request) {
    var errs []string
    sharder, errs := FormToSharder(r)

    // have we created a sharder correctly
    var created bool = false

    // if we had no errors from FormToSharder, we will
    // attempt to save our data to Cassandra
    if len(errs) == 0 {
        fmt.Println("creating a new sharder")
        // write data to Cassandra
        if err := session.Query(`
            INSERT INTO block (block_hash, prev_block_hash, block_signature, miner_id, timestamp, round) VALUES (?, ?, ?, ?, ?, ?)`,
            sharder.Block_hash, sharder.Prev_block_hash, sharder.Block_signature, sharder.Miner_id, sharder.Timestamp, sharder.Round).Exec(); err != nil {
            errs = append(errs, err.Error())
        } else {
            created = true
        }
    }

    if created {
        fmt.Println("Data inserted")
    } else {
        fmt.Println("errors", errs)
        json.NewEncoder(w).Encode(ErrorResponse{Errors: errs})
    }
}
