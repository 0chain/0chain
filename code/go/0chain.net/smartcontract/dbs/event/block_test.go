package event

import (
	"fmt"
	"os"
	"testing"
	"time"

	"0chain.net/core/config"
	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/stretchr/testify/require"
)

func TestAddBlock(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDbWithoutWorker(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	block := Block{}
	err = eventDb.addOrUpdateBlock(block)
	require.NoError(t, err, "Error while inserting Block to event Database")
	var count int64
	eventDb.Get().Table("blocks").Count(&count)
	require.Equal(t, int64(1), count, "Block is not inserted")
	err = eventDb.Drop()
	require.NoError(t, err)
}

func TestFindBlock(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDbWithoutWorker(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	defer func() {
		_ = eventDb.Drop()
	}()
	require.NoError(t, err)

	block := Block{
		UpdatableModel: model.UpdatableModel{ID: 1},
		Hash:           "test",
	}
	err = eventDb.addOrUpdateBlock(block)
	require.NoError(t, err, "Error while inserting Block to event Database")
	gotBlock, err := eventDb.GetBlockByHash("test")

	// To ignore createdAt and updatedAt
	block.CreatedAt = gotBlock.CreatedAt
	block.UpdatedAt = gotBlock.UpdatedAt
	require.Equal(t, block, gotBlock, "Block not getting inserted")

	block2 := Block{
		UpdatableModel: model.UpdatableModel{ID: 2},
		Hash:           "test2",
	}
	err = eventDb.addOrUpdateBlock(block2)
	require.NoError(t, err, "Error while inserting Block to event Database")
	gotBlocks, err := eventDb.GetBlocksByBlockNumbers(0, 1, common.Pagination{Limit: 20, IsDescending: true})
	if len(gotBlocks) != 2 {
		require.Error(t, fmt.Errorf("got %v blocks but expected 2", len(gotBlocks)))
	}
}

func TestGetRoundFromTime(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDbWithoutWorker(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	block := Block{
		UpdatableModel: model.UpdatableModel{CreatedAt: time.Now()},
		Hash:           "test",
	}
	err = eventDb.addOrUpdateBlock(block)
	require.NoError(t, err, "Error while inserting Block to event Database")
	_, err = eventDb.GetRoundFromTime(time.Now(), false)
	require.NoError(t, err, "Error while getting rounds from DB")
}

func TestGetBlocks(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()

	if err := eventDb.addPartition(0, "user_aggregates"); err != nil {
		t.Error()
	}

	block := Block{
		Hash:           "blockHash",
		Round:          7,
		IsFinalised:    true,
	}
	err := eventDb.addOrUpdateBlock(block)
	require.NoError(t, err, "Error while inserting Block to event Database")

	block2 := Block{
		Hash:           "blockHash_unf",
		Round:          8,
		IsFinalised:    false,
	}

	err = eventDb.addOrUpdateBlock(block2)
	require.NoError(t, err, "Error while inserting Block to event Database")

	blocks, err := eventDb.GetBlocks(common.Pagination{Limit: 10})
	require.NoError(t, err, "Error while getting blocks from DB")

	require.Equal(t, 1, len(blocks), "found more than one block")
	require.Equal(t, "blockHash", blocks[0].Hash, "found wrong block")
	require.Equal(t, int64(7), blocks[0].Round, "found wrong block")
	require.Equal(t, true, blocks[0].IsFinalised, "found wrong block")
}