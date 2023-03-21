package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

const (
	ethereumAddress       = "mock ethereum address"
	hash                  = "mock txn hash"
	nonce           int64 = 0
)

func TestBurnTicketEvent(t *testing.T) {
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

	eventDb, err := NewEventDb(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	burnTicket1 := BurnTicket{
		EthereumAddress: ethereumAddress,
		Hash:            hash,
		Nonce:           nonce,
	}

	err = eventDb.addBurnTicket(burnTicket1)
	require.NoError(t, err, "Error while inserting BurnTicket to event Database")

	eventDb.Get().Table("burn_tickets").Count(&count)
	require.Equal(t, int64(1), count, "BurnTicket not getting inserted")

	burnTickets, err := eventDb.GetBurnTickets(ethereumAddress)
	require.NoError(t, err, "Error while fetching burn tickets by ethereumAddress")
	require.Len(t, burnTickets, 1)

	burnTicket := burnTickets[0]
	require.Equal(t, ethereumAddress, burnTicket.EthereumAddress, "Fetched invalid BurnTicket")
	require.Equal(t, hash, burnTicket.Hash, "Fetched invalid BurnTicket")
	require.Equal(t, nonce, burnTicket.Nonce, "Fetched invalid BurnTicket")

	burnTicket2 := BurnTicket{
		EthereumAddress: ethereumAddress,
		Hash:            hash,
		Nonce:           nonce + 1,
	}

	err = eventDb.addBurnTicket(burnTicket2)
	require.Error(t, err, "Error while processing repeatable hash, inserting BurnTicket to event Database")

	eventDb.Get().Table("burn_tickets").Count(&count)
	require.Equal(t, int64(1), count, "BurnTicket gets inserted")

	burnTicket3 := BurnTicket{
		EthereumAddress: ethereumAddress,
		Hash:            hash + hash,
		Nonce:           nonce + 1,
	}

	err = eventDb.addBurnTicket(burnTicket3)
	require.NoError(t, err, "Error while inserting BurnTicket to event Database")

	eventDb.Get().Table("burn_tickets").Count(&count)
	require.Equal(t, int64(2), count, "BurnTicket not getting inserted")

	burnTickets, err = eventDb.GetBurnTickets(ethereumAddress)
	require.NoError(t, err, "Error while fetching burn tickets by ethereumAddress")
	require.Len(t, burnTickets, 2)

	burnTicket = burnTickets[1]
	require.Equal(t, ethereumAddress, burnTicket.EthereumAddress, "Fetched invalid BurnTicket")
	require.Equal(t, hash+hash, burnTicket.Hash, "Fetched invalid BurnTicket")
	require.Equal(t, nonce+1, burnTicket.Nonce, "Fetched invalid BurnTicket")

	burnTicket4 := BurnTicket{
		EthereumAddress: ethereumAddress,
		Hash:            hash + hash + hash,
		Nonce:           nonce + 1,
	}

	err = eventDb.addBurnTicket(burnTicket4)
	require.Error(t, err, "Error while processing repeatable ethereum address and nonce, inserting BurnTicket to event Database")

	eventDb.Get().Table("burn_tickets").Count(&count)
	require.Equal(t, int64(2), count, "BurnTicket gets inserted")
}
