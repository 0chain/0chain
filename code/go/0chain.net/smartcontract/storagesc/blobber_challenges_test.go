package storagesc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlobberChallenges_removeChallenges(t *testing.T) {

	var tt = []struct {
		name             string
		initChallengeIDs []string

		removeIDs []string

		expectChallengeIDs []BlobOpenChallenge
	}{
		{
			name:             "remove one challenge",
			initChallengeIDs: []string{"c1", "c2", "c3"},
			removeIDs:        []string{"c1"},
			expectChallengeIDs: []BlobOpenChallenge{
				BlobOpenChallenge{ID: "c2"},
				BlobOpenChallenge{ID: "c3"},
			},
		},
		{
			name:               "remove all challenges",
			initChallengeIDs:   []string{"c1", "c2", "c3"},
			removeIDs:          []string{"c1", "c2", "c3"},
			expectChallengeIDs: []BlobOpenChallenge{},
		},
		{
			name:             "remove multiple challenges",
			initChallengeIDs: []string{"c1", "c2", "c3", "c4", "c5"},
			removeIDs:        []string{"c2", "c5"},
			expectChallengeIDs: []BlobOpenChallenge{
				BlobOpenChallenge{ID: "c1"},
				BlobOpenChallenge{ID: "c3"},
				BlobOpenChallenge{ID: "c4"},
			},
		},
		{
			name:             "remove no challenge",
			initChallengeIDs: []string{"c1", "c2", "c3"},
			removeIDs:        []string{},
			expectChallengeIDs: []BlobOpenChallenge{
				BlobOpenChallenge{ID: "c1"},
				BlobOpenChallenge{ID: "c2"},
				BlobOpenChallenge{ID: "c3"},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			bc := BlobberChallenges{
				OpenChallenges: make([]BlobOpenChallenge, len(tc.initChallengeIDs)),
				ChallengesMap:  make(map[string]struct{}, len(tc.initChallengeIDs)),
			}

			for i, id := range tc.initChallengeIDs {
				bc.OpenChallenges[i] = BlobOpenChallenge{ID: id}
			}

			for _, id := range tc.initChallengeIDs {
				bc.ChallengesMap[id] = struct{}{}
			}

			bc.removeChallenges(tc.removeIDs)

			require.Equal(t, tc.expectChallengeIDs, bc.OpenChallenges)

			require.Equal(t, len(bc.OpenChallenges), len(bc.ChallengesMap))

			for _, oc := range bc.OpenChallenges {
				_, ok := bc.ChallengesMap[oc.ID]
				require.True(t, ok)
			}
		})
	}
}
