package event

import (
	"testing"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
)

func TestUpdateProvidersHealthCheck(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	err := edb.addBlobbers([]Blobber{
		{
			Provider: Provider{ID: "one"},
			BaseURL:  "one.com",
		}, {
			Provider: Provider{ID: "two"},
			BaseURL:  "two.com",
		},
	})

	a, err := edb.GetBlobberCount()
	a = a
	var blobbers0 []Blobber
	edb.Get().Find(&blobbers0)
	res := edb.Get().Model(Blobber{}).Find(&blobbers0)
	res = res
	require.Equal(t, len(blobbers0), 2)

	//func (edb *EventDb) updateProvidersHealthCheck(updates []dbs.DbHealthCheck, tableName ProviderTable) error
	updates := []dbs.DbHealthCheck{
		{
			ID:              "one",
			LastHealthCheck: 37,
			Downtime:        11,
		},
	}

	err = edb.updateProvidersHealthCheck(updates, "blobbers")
	require.NoError(t, err)

	var blobbers []Blobber
	edb.Get().Find(&blobbers)
	require.Equal(t, len(blobbers), 2)

}
