package event

import (
	"fmt"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/stakepool/spenum"
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

	eventDb, err := NewEventDb(access, config.DbSettings{})
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

	err = eventDb.addOrUpdateUsers([]User{user1})
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
	err = eventDb.addOrUpdateUsers([]User{user1})
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
	err = eventDb.addOrUpdateUsers([]User{user2})
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

func prepareEventDB(t *testing.T) (*EventDb, func()) {
	access := config.DbAccess{
		Enabled:         true,
		Name:            "crud",
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
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	return eventDb, func() {
		eventDb.Close()
	}
}

func TestAddAndUpdateUsersEvent(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	eventDb, closeDB := prepareEventDB(t)
	defer closeDB()

	// create new users
	users := make([]User, 10)
	for i := 0; i < 10; i++ {
		users[i] = User{
			UserID:  fmt.Sprintf("u_%v", i),
			TxnHash: fmt.Sprintf("hash_%v", i),
			Balance: currency.Coin(i),
			Nonce:   int64(i),
			Round:   int64(i),
		}
	}

	err := eventDb.addOrUpdateUsers(users)
	require.NoError(t, err, "Error while inserting Users to event Database")

	for i := 0; i < 10; i++ {
		u, err := eventDb.GetUser(fmt.Sprintf("u_%v", i))
		require.NoError(t, err)
		require.Equal(t, users[i].Balance, u.Balance)
		require.Equal(t, users[i].Nonce, u.Nonce)
		require.Equal(t, users[i].TxnHash, u.TxnHash)
		require.Equal(t, users[i].Round, u.Round)
	}

	// update users
	for i := 0; i < 10; i++ {
		users[i] = User{
			UserID:  fmt.Sprintf("u_%v", i),
			TxnHash: fmt.Sprintf("hash_%v", i),
			Balance: currency.Coin(i * 100),
			Nonce:   int64(i + 100),
			Round:   int64(i + 100),
		}
	}

	err = eventDb.addOrUpdateUsers(users)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		u, err := eventDb.GetUser(fmt.Sprintf("u_%v", i))
		require.NoError(t, err)
		require.Equal(t, users[i].Balance, u.Balance)
		require.Equal(t, users[i].Nonce, u.Nonce)
		require.Equal(t, users[i].TxnHash, u.TxnHash)
		require.Equal(t, users[i].Round, u.Round)
	}

	users = make([]User, 10)

	// add and update
	for i := 5; i < 15; i++ {
		users[i-5] = User{
			UserID:  fmt.Sprintf("u_%v", i),
			TxnHash: fmt.Sprintf("hash_%v", i),
			Balance: currency.Coin(i * 150),
			Nonce:   int64(i + 150),
			Round:   int64(i + 150),
		}
	}

	err = eventDb.addOrUpdateUsers(users)
	require.NoError(t, err)

	for i := 5; i < 15; i++ {
		u, err := eventDb.GetUser(fmt.Sprintf("u_%v", i))
		require.NoError(t, err)
		require.Equal(t, users[i-5].Balance, u.Balance)
		require.Equal(t, users[i-5].Nonce, u.Nonce)
		require.Equal(t, users[i-5].TxnHash, u.TxnHash)
		require.Equal(t, users[i-5].Round, u.Round)
	}
}

func TestAddAndUpdateStakePoolRewards(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	//	eventDb, closeDB := prepareEventDB(t)
	//	defer closeDB()

	// create new users
	miners := make([]Miner, 10)
	for i := 0; i < 10; i++ {
		miners[i] = Miner{
			Provider: Provider{
				ID: fmt.Sprintf("m_%v", i),
				Rewards: ProviderRewards{
					ProviderID:   fmt.Sprintf("m_%v", i),
					Rewards:      currency.Coin((i + 1) * 10),
					TotalRewards: currency.Coin((i + 1) * 1000),
				},
			},
		}
	}

	//err := rewardProvider(eventDb, "miner_id", "miners", miners)
	//require.NoError(t, err)
}

func TestUpdateStakePoolDelegateRewards(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	eventDb, closeDB := prepareEventDB(t)
	defer closeDB()

	// create new users
	var miners []DelegatePool
	for i := 0; i < 10; i++ {
		miners = append(miners, DelegatePool{
			ProviderID:   fmt.Sprintf("pd_%v", i),
			ProviderType: spenum.Miner,
			DelegateID:   fmt.Sprintf("p_%v", i),
			PoolID:       fmt.Sprintf("p_%v", i),
			Reward:       currency.Coin((i + 1) * 10),
			TotalReward:  currency.Coin((i + 1) * 1000),
		})
	}

	err := rewardProviderDelegates(eventDb, miners)
	require.NoError(t, err)
}
