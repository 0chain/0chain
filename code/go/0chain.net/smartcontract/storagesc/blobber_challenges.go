package storagesc

import (
	"time"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

//msgp:ignore BlobberChallenges
//go:generate msgp -io=false -tests=false -unexported -v

// BlobOpenChallenge records the open challenges info of blobber
type BlobOpenChallenge struct {
	ID        string           `json:"id"`
	CreatedAt common.Timestamp `json:"created_at"`
}

// BlobberChallenges collects all the open challenges of a blobber that will
// be used for `/openchallenges` endpoint
type BlobberChallenges struct {
	BlobberID                string              `json:"blobber_id"`
	LatestCompletedChallenge *StorageChallenge   `json:"lastest_completed_challenge"`
	OpenChallenges           []BlobOpenChallenge `json:"challenges"`
	ChallengesMap            map[string]struct{} `json:"-" msg:"-"`
}

func (bcs *BlobberChallenges) GetKey(globalKey string) datastore.Key {
	return globalKey + ":blobberchallenges:" + bcs.BlobberID
}

func (bcs *BlobberChallenges) load(state state.StateContextI, globalKey string) error {
	return state.GetTrieNode(bcs.GetKey(globalKey), bcs)
}

func (bcs *BlobberChallenges) save(state state.StateContextI, globalKey string) error {
	_, err := state.InsertTrieNode(bcs.GetKey(globalKey), bcs)
	return err
}

func (bcs *BlobberChallenges) addChallenge(challengeID string, createdAt common.Timestamp) bool {
	if bcs.ChallengesMap == nil {
		bcs.ChallengesMap = make(map[string]struct{})
	}

	if _, ok := bcs.ChallengesMap[challengeID]; !ok {
		bcs.OpenChallenges = append(bcs.OpenChallenges,
			BlobOpenChallenge{
				ID:        challengeID,
				CreatedAt: createdAt,
			})
		bcs.ChallengesMap[challengeID] = struct{}{}
		return true
	}
	return false
}

func (bcs *BlobberChallenges) removeChallenge(challenge *StorageChallenge) bool {
	if _, ok := bcs.ChallengesMap[challenge.ID]; !ok {
		return false
	}

	delete(bcs.ChallengesMap, challenge.ID)
	for i, oc := range bcs.OpenChallenges {
		if oc.ID == challenge.ID {

			bcs.OpenChallenges = append(bcs.OpenChallenges[:i], bcs.OpenChallenges[i+1:]...)
			bcs.LatestCompletedChallenge = challenge
			return true
		}
	}

	return false
}

func (bcs *BlobberChallenges) removeChallenges(ids []string) {
	deleteMap := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		deleteMap[id] = struct{}{}
	}

	newOpenChallenges := make([]BlobOpenChallenge, 0, len(bcs.OpenChallenges))
	for _, oc := range bcs.OpenChallenges {
		if _, ok := deleteMap[oc.ID]; !ok {
			newOpenChallenges = append(newOpenChallenges, oc)
			continue
		}

		delete(bcs.ChallengesMap, oc.ID)
	}

	bcs.OpenChallenges = newOpenChallenges
}

func isChallengeExpired(now, createdAt common.Timestamp, challengeCompletionTime time.Duration) bool {
	return createdAt+common.ToSeconds(challengeCompletionTime) <= now
}

// GetOpenChallengesNoExpire returns open challenges ids that are not expired
func (bcs *BlobberChallenges) GetOpenChallengesNoExpire(tm common.Timestamp, challengeCompleteTime time.Duration) []BlobOpenChallenge {
	for i, oc := range bcs.OpenChallenges {
		if !isChallengeExpired(tm, oc.CreatedAt, challengeCompleteTime) {
			return bcs.OpenChallenges[i:]
		}
	}

	return nil
}

type blobberChallengeDecode BlobberChallenges

func (bcs *BlobberChallenges) MarshalMsg(o []byte) ([]byte, error) {
	d := blobberChallengeDecode(*bcs)
	return d.MarshalMsg(o)
}

func (bcs *BlobberChallenges) UnmarshalMsg(data []byte) ([]byte, error) {
	d := &blobberChallengeDecode{}
	o, err := d.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	*bcs = BlobberChallenges(*d)

	bcs.ChallengesMap = make(map[string]struct{})
	for _, challenge := range bcs.OpenChallenges {
		bcs.ChallengesMap[challenge.ID] = struct{}{}
	}
	return o, nil
}
