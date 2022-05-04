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

		expectChallengeIDs []string
	}{
		{
			name:               "remove one challenge",
			initChallengeIDs:   []string{"c1", "c2", "c3"},
			removeIDs:          []string{"c1"},
			expectChallengeIDs: []string{"c2", "c3"},
		},
		{
			name:               "remove all challenges",
			initChallengeIDs:   []string{"c1", "c2", "c3"},
			removeIDs:          []string{"c1", "c2", "c3"},
			expectChallengeIDs: []string{},
		},
		{
			name:               "remove multiple challenges",
			initChallengeIDs:   []string{"c1", "c2", "c3", "c4", "c5"},
			removeIDs:          []string{"c2", "c5"},
			expectChallengeIDs: []string{"c1", "c3", "c4"},
		},
		{
			name:               "remove no challenge",
			initChallengeIDs:   []string{"c1", "c2", "c3"},
			removeIDs:          []string{},
			expectChallengeIDs: []string{"c1", "c2", "c3"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			bc := BlobberChallenges{
				ChallengeIDs:   tc.initChallengeIDs,
				ChallengeIDMap: make(map[string]struct{}, len(tc.initChallengeIDs)),
			}

			for _, id := range tc.initChallengeIDs {
				bc.ChallengeIDMap[id] = struct{}{}
			}

			bc.removeChallenges(tc.removeIDs)

			require.Equal(t, tc.expectChallengeIDs, bc.ChallengeIDs)

			require.Equal(t, len(bc.ChallengeIDs), len(bc.ChallengeIDMap))

			for _, id := range bc.ChallengeIDs {
				_, ok := bc.ChallengeIDMap[id]
				require.True(t, ok)
			}
		})
	}
}
