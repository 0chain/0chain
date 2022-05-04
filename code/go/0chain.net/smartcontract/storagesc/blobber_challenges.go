package storagesc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

//msgp:ignore BlobberChallenges
//go:generate msgp -io=false -tests=false -unexported -v

// BlobberChallenges collects all the open challenges of a blobber that will
// be used for `/openchallenges` endpoint
type BlobberChallenges struct {
	BlobberID                string              `json:"blobber_id"`
	LatestCompletedChallenge *StorageChallenge   `json:"lastest_completed_challenge"`
	ChallengeIDs             []string            `json:"challenge_ids"`
	ChallengeIDMap           map[string]struct{} `json:"-" msg:"-"`
}

func (sn *BlobberChallenges) GetKey(globalKey string) datastore.Key {
	return globalKey + ":blobberchallenge:" + sn.BlobberID
}

func (sn *BlobberChallenges) load(state state.StateContextI, globalKey string) error {
	return state.GetTrieNode(sn.GetKey(globalKey), sn)
}

func (sn *BlobberChallenges) save(state state.StateContextI, globalKey string) error {
	_, err := state.InsertTrieNode(sn.GetKey(globalKey), sn)
	return err
}

func (sn *BlobberChallenges) addChallenge(challengeID string) bool {
	if sn.ChallengeIDMap == nil {
		sn.ChallengeIDMap = make(map[string]struct{})
	}

	if _, ok := sn.ChallengeIDMap[challengeID]; !ok {
		sn.ChallengeIDs = append(sn.ChallengeIDs, challengeID)
		sn.ChallengeIDMap[challengeID] = struct{}{}
		return true
	}
	return false
}

func (sn *BlobberChallenges) removeChallenge(challenge *StorageChallenge) bool {
	if _, ok := sn.ChallengeIDMap[challenge.ID]; !ok {
		return false
	}

	delete(sn.ChallengeIDMap, challenge.ID)
	for i, id := range sn.ChallengeIDs {
		if id == challenge.ID {

			sn.ChallengeIDs = append(sn.ChallengeIDs[:i], sn.ChallengeIDs[i+1:]...)
			sn.LatestCompletedChallenge = challenge
			return true
		}
	}

	return false
}

func (sn *BlobberChallenges) removeChallenges(ids []string) {
	deleteMap := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		deleteMap[id] = struct{}{}
	}

	cids := make([]string, 0, len(sn.ChallengeIDs))
	for _, cid := range sn.ChallengeIDs {
		if _, ok := deleteMap[cid]; !ok {
			cids = append(cids, cid)
			continue
		}

		delete(sn.ChallengeIDMap, cid)
	}

	sn.ChallengeIDs = cids
}

type blobberChallengeDecode BlobberChallenges

func (sn *BlobberChallenges) MarshalMsg(o []byte) ([]byte, error) {
	d := blobberChallengeDecode(*sn)
	return d.MarshalMsg(o)
}

func (sn *BlobberChallenges) UnmarshalMsg(data []byte) ([]byte, error) {
	d := &blobberChallengeDecode{}
	o, err := d.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	*sn = BlobberChallenges(*d)

	sn.ChallengeIDMap = make(map[string]struct{})
	for _, challenge := range sn.ChallengeIDs {
		sn.ChallengeIDMap[challenge] = struct{}{}
	}
	return o, nil
}
