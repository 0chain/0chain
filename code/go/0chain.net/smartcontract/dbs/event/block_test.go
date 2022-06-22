package event

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"0chain.net/chaincore/config"
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
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	block := Block{}
	err = eventDb.addBlock(block)
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
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	defer func() {
		_ = eventDb.Drop()
	}()
	require.NoError(t, err)

	block := Block{
		Model: gorm.Model{ID: 1},
		Hash:  "test",
	}
	err = eventDb.addBlock(block)
	require.NoError(t, err, "Error while inserting Block to event Database")
	gotBlock, err := eventDb.GetBlocksByHash("test")

	// To ignore createdAt and updatedAt
	block.CreatedAt = gotBlock.CreatedAt
	block.UpdatedAt = gotBlock.UpdatedAt
	require.Equal(t, block, gotBlock, "Block not getting inserted")

	block2 := Block{
		Model: gorm.Model{ID: 2},
		Hash:  "test2",
	}
	err = eventDb.addBlock(block2)
	require.NoError(t, err, "Error while inserting Block to event Database")
	gotBlocks, err := eventDb.GetBlocks(Pagination{0, 20, true})
	if len(gotBlocks) != 2 {
		require.Error(t, fmt.Errorf("got %v blocks but expected 2", len(gotBlocks)))
	}
}
