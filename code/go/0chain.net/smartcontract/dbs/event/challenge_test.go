package event

import (
	"encoding/json"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
)

func TestChallenges(t *testing.T) {
	//t.Skip("only for local debugging, requires local postgresql")
	access := dbs.DbAccess{
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
	err = eventDb.drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	blobber1 := Blobber{
		BlobberID: "one",
		BaseURL:   "one url",
	}
	data, err := json.Marshal(&blobber1)
	require.NoError(t, err)
	eventAddBlobber := Event{
		BlockNumber: 2,
		TxHash:      "tx hash1",
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteBlobber),
		Index:       blobber1.BlobberID,
		Data:        string(data),
	}

	challenge1 := Challenge{
		BlobberID:   "one",
		ChallengeID: "first",
		BlobberUrl:  "one url",
		Validators: []ValidationNode{
			{ValidatorID: "val one"}, {ValidatorID: "val two"},
		},
	}

	challenge2 := Challenge{
		BlobberID:   "two",
		ChallengeID: "second",
		BlobberUrl:  "two url",
		Validators: []ValidationNode{
			{ValidatorID: "val two one"}, {ValidatorID: "val two two"},
		},
	}

	data, err = json.Marshal(&challenge1)
	require.NoError(t, err)
	eventAddCh := Event{
		BlockNumber: 2,
		TxHash:      "tx hash2",
		Type:        int(TypeStats),
		Tag:         int(TagAddChallenge),
		Index:       challenge1.ChallengeID,
		Data:        string(data),
	}

	data2, err := json.Marshal(&challenge2)
	require.NoError(t, err)
	eventAddCh2 := Event{
		BlockNumber: 2,
		TxHash:      "tx hash3",
		Type:        int(TypeStats),
		Tag:         int(TagAddChallenge),
		Index:       challenge2.ChallengeID,
		Data:        string(data2),
	}
	require.NoError(t, err)

	events := []Event{eventAddBlobber, eventAddCh, eventAddCh2}
	eventDb.AddEvents(events)

	ch, err := eventDb.GetChallenge(challenge1.ChallengeID)
	require.NoError(t, err)
	require.EqualValues(t, len(challenge1.Validators), len(ch.Validators))
	ch = ch

	bc, err := eventDb.GetBlobberChallenges("one")
	require.NoError(t, err)
	require.EqualValues(t, len(bc.Challenges), 1)
	require.EqualValues(t, len(bc.Challenges[0].Validators), 2)

	deleteEvent := Event{
		BlockNumber: 3,
		TxHash:      "tx hash4",
		Type:        int(TypeStats),
		Tag:         int(TagDeleteChallenge),
		Index:       challenge1.ChallengeID,
		Data:        challenge1.ChallengeID,
	}
	eventDb.AddEvents([]Event{deleteEvent})
	_, err = eventDb.GetChallenge(challenge1.ChallengeID)
	require.Error(t, err)
}
