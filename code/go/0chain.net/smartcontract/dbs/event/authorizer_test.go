package event

import (
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestAuthorizers(t *testing.T) {
	t.Skip("only for local debugging, requires local postgres")

	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            "zchain_user",
		Password:        "zchian",
		Host:            "localhost",
		Port:            "5432",
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDb(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	authorizer_1 := Authorizer{
		URL:             "http://localhost:8080",
		Latitude:        0.0,
		Longitude:       0.0,
		Provider: Provider{
			ID:             encryption.Hash("mockAuthorizer_" + strconv.Itoa(0)),
			DelegateWallet: "delegate wallet",
			MinStake:       currency.Coin(53),
			MaxStake:       currency.Coin(57),
			NumDelegates:   59,
			ServiceCharge:  61.0,
			LastHealthCheck: common.Timestamp(time.Now().Unix()),
		},
	}

	authorizer_2 := Authorizer{
		URL:             "http://localhost:8888",
		Latitude:        1.0,
		Longitude:       1.0,
		Provider: Provider{
			ID:             encryption.Hash("mockAuthorizer_" + strconv.Itoa(1)),
			DelegateWallet: "delegate wallet",
			MinStake:       currency.Coin(52),
			MaxStake:       currency.Coin(57),
			NumDelegates:   60,
			ServiceCharge:  50.0,
			LastHealthCheck: common.Timestamp(time.Now().Unix()),
		},
	}

	err = eventDb.AddAuthorizer(&authorizer_1)
	require.NoError(t, err, "Error while inserting Authorizer to event Database")

	var count int64
	eventDb.Get().Table("authorizers").Count(&count)
	require.Equal(t, int64(1), count, "Authorizer not getting inserted")

	err = eventDb.AddAuthorizer(&authorizer_2)
	require.NoError(t, err, "Error while inserting Authorizer to event Database")

	eventDb.Get().Table("authorizers").Count(&count)
	require.Equal(t, int64(2), count, "Authorizer not getting inserted")

	_, err = eventDb.GetValidatorByValidatorID(authorizer_1.ID)
	require.NoError(t, err, "Error while getting Authorizer from event Database")

	_, err = authorizer_2.exists(eventDb)
	require.NoError(t, err, "Error while checking if Authorizer exists in event Database")

	err = eventDb.Drop()
	require.NoError(t, err)

}
