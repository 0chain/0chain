package event

import (
	"strconv"
	"testing"

	"github.com/0chain/common/core/currency"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestValidatorNode(t *testing.T) {
	t.Run("test addOrOverwriteValidators", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		vn := Validator{
			BaseUrl: "http://localhost:8080",
			Provider: Provider{
				ID:         encryption.Hash("mockValidator_" + strconv.Itoa(0)),
				TotalStake: 100,
	
				DelegateWallet: "delegate wallet",
				MinStake:       currency.Coin(53),
				MaxStake:       currency.Coin(57),
				NumDelegates:   59,
				ServiceCharge:  61.0,
			},
		}
		err := eventDb.addOrOverwriteValidators([]Validator{vn})
		require.NoError(t, err, "Error while inserting Validation Node to event Database")
	
		var count int64
		eventDb.Get().Table("validators").Count(&count)
		require.Equal(t, int64(1), count, "Validator not getting inserted")
	
		vnFromDb, err := eventDb.GetValidatorByValidatorID(vn.ID)
		require.NoError(t, err, "Error while getting Validation Node from event Database")
		require.Equal(t, vn.BaseUrl, vnFromDb.BaseUrl)
		require.Equal(t, vn.TotalStake, vnFromDb.TotalStake)
		require.Equal(t, vn.DelegateWallet, vnFromDb.DelegateWallet)
		require.Equal(t, vn.DelegateWallet, vnFromDb.DelegateWallet)
		require.Equal(t, vn.MinStake, vnFromDb.MinStake)
		require.Equal(t, vn.MaxStake, vnFromDb.MaxStake)
		require.Equal(t, vn.MaxStake, vnFromDb.MaxStake)
		require.Equal(t, vn.NumDelegates, vnFromDb.NumDelegates)
		require.Equal(t, vn.ServiceCharge, vnFromDb.ServiceCharge)
	})

	t.Run("test updateValidators", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		vn := Validator{
			BaseUrl: "http://localhost:8080",
			Provider: Provider{
				ID:         encryption.Hash("mockValidator_" + strconv.Itoa(0)),
				TotalStake: 100,
	
				DelegateWallet: "delegate wallet",
				MinStake:       currency.Coin(53),
				MaxStake:       currency.Coin(57),
				NumDelegates:   59,
				ServiceCharge:  61.0,
			},
		}
		err := eventDb.addOrOverwriteValidators([]Validator{vn})
		require.NoError(t, err, "Error while inserting Validation Node to event Database")

		vnUpdated := Validator{
			BaseUrl: "http://localhost:8082",
			Provider: Provider{
				ID:         vn.ID,
				TotalStake: 102,
	
				DelegateWallet: "delegate wallet edited",
				MinStake:       currency.Coin(50),
				MaxStake:       currency.Coin(51),
				NumDelegates:   60,
				ServiceCharge:  62.03,
			},
		}

		err = eventDb.updateValidators([]Validator{ vnUpdated })

		require.NoError(t, err, "Error while updating Validation Node to event Database")

		vnFromDb, err := eventDb.GetValidatorByValidatorID(vn.ID)
		require.NoError(t, err, "Error while getting Validation Node from event Database")

		require.Equal(t, vnUpdated.BaseUrl, vnFromDb.BaseUrl)
		require.Equal(t, vnUpdated.TotalStake, vnFromDb.TotalStake)
		require.Equal(t, vnUpdated.DelegateWallet, vnFromDb.DelegateWallet)
		require.Equal(t, vnUpdated.MinStake, vnFromDb.MinStake)
		require.Equal(t, vnUpdated.MaxStake, vnFromDb.MaxStake)
		require.Equal(t, vnUpdated.NumDelegates, vnFromDb.NumDelegates)
		require.Equal(t, vnUpdated.ServiceCharge, vnFromDb.ServiceCharge)
	})
}
