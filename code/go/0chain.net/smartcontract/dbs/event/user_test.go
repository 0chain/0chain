package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

var (
	clientID       = "mock client ID"
	txnHash        = "mock txn hash"
	initialBalance = 10
	count          int64
	clientID2      = clientID + " 2"
)

func TestUserEvent(t *testing.T) {
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

	user1 := User{
		UserID:  clientID,
		TxnHash: txnHash,
		Balance: currency.Coin(initialBalance),
		Round:   3,
		Nonce:   1,
	}

	err = eventDb.addOrOverwriteUser(user1)
	require.NoError(t, err, "Error while inserting User to event Database")

	eventDb.Get().Table("users").Count(&count)
	require.Equal(t, int64(1), count, "User not getting inserted")

	user, err := eventDb.GetUser(clientID)
	require.NoError(t, err, "Error while fetching user by clientID")
	require.Equal(t, clientID, user.UserID, "Fetched invalid User")
	require.Equal(t, txnHash, user.TxnHash, "Fetched invalid User")
	require.Equal(t, initialBalance, user.Balance, "Fetched invalid User")
	require.Equal(t, 1, user.Nonce, "Fetched invalid User")

	user1.Balance = user1.Balance + 1
	user1.Nonce = user1.Nonce + 1
	err = eventDb.addOrOverwriteUser(user1)
	require.NoError(t, err, "Error while inserting User to event Database")

	eventDb.Get().Table("users").Count(&count)
	require.Equal(t, int64(1), count, "User not getting overwritten")

	user, err = eventDb.GetUser(clientID)
	require.NoError(t, err, "Error while fetching user by clientID")
	require.Equal(t, clientID, user.UserID, "Fetched invalid User")
	require.Equal(t, txnHash, user.TxnHash, "Fetched invalid User")
	require.Equal(t, initialBalance+1, user.Balance, "Fetched invalid User")
	require.Equal(t, 2, user.Nonce, "Fetched invalid User")

	//clientID2 := u.UserID + " 2"
	user2 := User{
		UserID:  clientID2,
		TxnHash: txnHash + " 2",
		Balance: currency.Coin(initialBalance) - 1,
		Round:   10,
		Nonce:   1,
	}
	err = eventDb.addOrOverwriteUser(user2)
	require.NoError(t, err, "Error while inserting User to event Database")

	user, err = eventDb.GetUser(clientID2)
	require.NoError(t, err, "Error while fetching user by clientID")
	require.Equal(t, clientID2, user.UserID, "Fetched invalid User")
	require.Equal(t, 1, user.Nonce, "Fetched invalid User")

	eventDb.Get().Table("users").Count(&count)
	require.Equal(t, int64(2), count, "Should have two separate users in store")
	require.Equal(t, int64(3), count, "Just failing for testing purposes")

	err = eventDb.Drop()
	require.NoError(t, err)
}
