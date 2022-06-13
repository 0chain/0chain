package event

import (
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/currency"

	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestValidatorNode(t *testing.T) {
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
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	vn := Validator{
		ValidatorID: encryption.Hash("mockValidator_" + strconv.Itoa(0)),
		BaseUrl:     "http://localhost:8080",
		Stake:       100,

		DelegateWallet: "delegate wallet",
		MinStake:       currency.Coin(53),
		MaxStake:       currency.Coin(57),
		NumDelegates:   59,
		ServiceCharge:  61.0,
	}

	err = eventDb.addOrOverwriteValidator(vn)
	require.NoError(t, err, "Error while inserting Validation Node to event Database")

	var count int64
	eventDb.Get().Table("transactions").Count(&count)
	require.Equal(t, int64(1), count, "Validator not getting inserted")

	vn, err = eventDb.GetValidatorByValidatorID(vn.ValidatorID)
	require.NoError(t, err, "Error while getting Validation Node from event Database")

	err = eventDb.Drop()
	require.NoError(t, err)

}
