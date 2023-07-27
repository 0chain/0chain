package event

import (
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"gorm.io/gorm/clause"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/go-faker/faker/v4"
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
		URL:       "http://localhost:8080",
		Latitude:  0.0,
		Longitude: 0.0,
		Provider: Provider{
			ID:              encryption.Hash("mockAuthorizer_" + strconv.Itoa(0)),
			DelegateWallet:  "delegate wallet",
			NumDelegates:    59,
			ServiceCharge:   61.0,
			LastHealthCheck: common.Timestamp(time.Now().Unix()),
		},
	}

	authorizer_2 := Authorizer{
		URL:       "http://localhost:8888",
		Latitude:  1.0,
		Longitude: 1.0,
		Provider: Provider{
			ID:              encryption.Hash("mockAuthorizer_" + strconv.Itoa(1)),
			DelegateWallet:  "delegate wallet",
			NumDelegates:    60,
			ServiceCharge:   50.0,
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

	activeAuthorizers, err := eventDb.GetActiveAuthorizers()
	require.NoError(t, err, "Error while active Authorizer retrieval")
	require.Len(t, activeAuthorizers, 2)

	require.Equal(t, authorizer_1.ID, activeAuthorizers[0].ID)
	require.Equal(t, authorizer_2.ID, activeAuthorizers[1].ID)

	err = eventDb.Drop()
	require.NoError(t, err)

}

func Test_authorizerMintAndBurn(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	err := edb.Store.Get().Model(&Authorizer{}).Omit(clause.Associations).Create([]Authorizer{
		{
			Provider: Provider{
				ID: "auth1",
			},
			TotalMint: 0,
			TotalBurn: 0,
		},
		{
			Provider: Provider{
				ID: "auth2",
			},
			TotalMint: 0,
			TotalBurn: 0,
		},
	}).Error
	require.NoError(t, err)

	var (
		authorizersBefore, authorizersAfter []Authorizer
	)

	err = edb.Store.Get().Model(&Authorizer{}).Omit(clause.Associations).Order("id ASC").Find(&authorizersBefore).Error
	require.NoError(t, err)
	err = edb.updateAuthorizersTotalMint([]state.Mint{
		{
			ToClientID: "auth1",
			Amount:     20,
		},
		{
			ToClientID: "auth2",
			Amount:     200,
		},
	})
	require.NoError(t, err)
	err = edb.updateAuthorizersTotalBurn([]state.Burn{
		{
			Burner: "auth1",
			Amount: 5,
		},
		{
			Burner: "auth2",
			Amount: 50,
		},
	})
	require.NoError(t, err)

	err = edb.Store.Get().Model(&Authorizer{}).Omit(clause.Associations).Order("id ASC").Find(&authorizersAfter).Error
	require.NoError(t, err)
	require.Equal(t, authorizersBefore[0].TotalMint+20, authorizersAfter[0].TotalMint)
	require.Equal(t, authorizersBefore[0].TotalBurn+5, authorizersAfter[0].TotalBurn)
	require.Equal(t, authorizersBefore[1].TotalMint+200, authorizersAfter[1].TotalMint)
	require.Equal(t, authorizersBefore[1].TotalBurn+50, authorizersAfter[1].TotalBurn)
}

func buildMockAuthorizer(t *testing.T, ownerId string, pid string) Authorizer {
	var authorizer Authorizer
	err := faker.FakeData(&authorizer)
	require.NoError(t, err)

	authorizer.ID = pid
	authorizer.DelegateWallet = ownerId
	authorizer.IsKilled = false
	authorizer.IsShutdown = false
	authorizer.Rewards = ProviderRewards{}
	return authorizer
}