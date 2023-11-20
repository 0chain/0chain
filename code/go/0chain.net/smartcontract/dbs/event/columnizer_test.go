package event

import (
	"testing"

	"github.com/0chain/common/core/currency"
	"github.com/stretchr/testify/require"
)

func TestColumnizer(t *testing.T) {
	t.Run("test toSnakeCase", func(t *testing.T) {
		str := "NewField"
		snakeStr := toSnakeCase(str)
		require.Equal(t, "new_field", snakeStr, "toSnakeCase not working as expected for 'NewField'")

		str = "field"
		snakeStr = toSnakeCase(str)
		require.Equal(t, "field", snakeStr, "toSnakeCase not working as expected for 'field'")
	})

	t.Run("test Columnize", func(t *testing.T) {
		// Test invalid type
		cols, err := Columnize([]string{"a1", "a2", "a3"})
		require.Error(t, err)
		require.Equal(t, "columnize error: type invalid", err.Error())

		type AllocationHash struct {
			Hash string
			HashExpiration int64
		}
		type Provider struct {
			ProviderID string
		}
		type Allocation struct {
			AllocationID  string `gorm:"column:allocation_id_gorm;"`
			ParityShards  int
			Size          int64
			Price         float64
			Owner         string
			WritePriceMax currency.Coin
			Finalized     bool
			FileOptions   uint16
			Provider 	  *Provider
			Hash		  AllocationHash
			References	  []string
			User  User                    `gorm:"foreignKey:Owner;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
			Terms []AllocationBlobberTerm `json:"terms" gorm:"foreignKey:AllocationID;references:AllocationID"`

		}

		allocs := []Allocation{
			{
				AllocationID:  "allocation1",
				ParityShards:  10,
				Size:          1900000000,
				Price:         0.00098,
				Owner:         "owner1",
				WritePriceMax: currency.Coin(100),
				Finalized:     true,
				FileOptions:   63,
				References:   []string{"ref1", "ref2", "ref3"},
				Provider: &Provider{ProviderID: "provider1"},
				Hash: AllocationHash{Hash: "hash1", HashExpiration: 100},
			},
			{
				AllocationID:  "allocation2",
				ParityShards:  20,
				Size:          2900000000,
				Price:         0.000998,
				Owner:         "owner2",
				WritePriceMax: currency.Coin(200),
				Finalized:     false,
				FileOptions:   60,
				References:   []string{"ref4", "ref5", "ref6"},
				Provider: &Provider{ProviderID: "provider2"},
				Hash: AllocationHash{Hash: "hash2", HashExpiration: 200},
			},
			{
				AllocationID:  "allocation3",
				ParityShards:  30,
				Size:          3900000000,
				Price:         0.0009998,
				Owner:         "owner3",
				WritePriceMax: currency.Coin(300),
				Finalized:     true,
				FileOptions:   1,
				References:   []string{"ref7", "ref8", "ref9"},
				Provider: &Provider{ProviderID: "provider3"},
				Hash: AllocationHash{Hash: "hash3", HashExpiration: 300},
			},
		}

		cols, err = Columnize(allocs)
		require.NoError(t, err)
		require.Equal(t, 12, len(cols))

		// test gorm field > snake field
		colValues, ok := cols["allocation_id"]
		require.False(t, ok)
		require.Nil(t, colValues)
		colValues, ok = cols["allocation_id_gorm"]
		require.True(t, ok)
		require.Equal(t, []interface{}{"allocation1", "allocation2", "allocation3"}, colValues)

		// test other fields
		colValues, ok = cols["parity_shards"]
		require.True(t, ok)
		require.Equal(t, []interface{}{10, 20, 30}, colValues)

		// test snake case fields
		colValues, ok = cols["size"]
		require.True(t, ok)
		require.Equal(t, []interface{}{int64(1900000000), int64(2900000000), int64(3900000000)}, colValues)

		colValues, ok = cols["price"]
		require.True(t, ok)
		require.Equal(t, []interface{}{float64(0.00098), float64(0.000998), float64(0.0009998)}, colValues)

		colValues, ok = cols["owner"]
		require.True(t, ok)
		require.Equal(t, []interface{}{"owner1", "owner2", "owner3"}, colValues)

		colValues, ok = cols["write_price_max"]
		require.True(t, ok)
		require.Equal(t, []interface{}{currency.Coin(100), currency.Coin(200), currency.Coin(300)}, colValues)

		colValues, ok = cols["finalized"]
		require.True(t, ok)
		require.Equal(t, []interface{}{true, false, true}, colValues)

		colValues, ok = cols["file_options"]
		require.True(t, ok)
		require.Equal(t, []interface{}{uint16(63), uint16(60), uint16(1)}, colValues)

		// slice field
		colValues, ok = cols["references"]
		require.True(t, ok)
		require.Equal(t, []interface{}{[]string{"ref1", "ref2", "ref3"}, []string{"ref4", "ref5", "ref6"}, []string{"ref7", "ref8", "ref9"}}, colValues)
		
		// strcut field
		colValues, ok = cols["provider"]
		require.False(t, ok)
		require.Nil(t, colValues)
		colValues, ok = cols["provider_id"]
		require.True(t, ok)
		require.Equal(t, []interface{}{"provider1", "provider2", "provider3"}, colValues)
		
		// ptr field
		colValues, ok = cols["allocation_hash"]
		require.False(t, ok)
		require.Nil(t, colValues)
		colValues, ok = cols["hash"]
		require.True(t, ok)
		require.Equal(t, []interface{}{"hash1", "hash2", "hash3"}, colValues)
		colValues, ok = cols["hash_expiration"]
		require.True(t, ok)
		require.Equal(t, []interface{}{int64(100), int64(200), int64(300)}, colValues)


		// struct foreignKey
		colValues, ok = cols["user"]
		require.False(t, ok)
		require.Nil(t, colValues)

		// slice foreignKey
		colValues, ok = cols["terms"]
		require.False(t, ok)
		require.Nil(t, colValues)
	})
}
