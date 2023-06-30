package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/encryption"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

type AllocOpenChallenge struct {
	// TODO: remember to update index when allocation updated to add or remove blobbers
	BlobberIndex int8   `json:"index" msg:"i"`      // blobber index in an allocation
	CreatedAt    int64  `json:"created_at" msg:"t"` // timestamp when challenge created
	ChallengeID  string `json:"challenge_id" msg:"c"`
}

type AllocationChallenges struct {
	AllocationID   string                `json:"allocation_id" msg:"-"`
	OpenChallenges []*AllocOpenChallenge `json:"open_challenges" msg:"o"`
	//ChallengeMap   map[string]*AllocOpenChallenge `json:"-" msg:"-"`
}

func (acs *AllocationChallenges) GetKey(globalKey string) datastore.Key {
	return encryption.Hash(globalKey + ":allocation_challenges:" + acs.AllocationID)
}

func (acs *AllocationChallenges) find(ID string) bool {
	for _, challenge := range acs.OpenChallenges {
		if challenge.ChallengeID == ID {
			return true
		}
	}
	return false
}

//func (acs *AllocationChallenges) MarshalMsg(b []byte) ([]byte, error) {
//	d := allocationChallengesDecoder(*acs)
//	return d.MarshalMsg(b)
//}

//func (acs *AllocationChallenges) UnmarshalMsg(b []byte) ([]byte, error) {
//	d := &allocationChallengesDecoder{}
//	v, err := d.UnmarshalMsg(b)
//	if err != nil {
//		return nil, err
//	}
//
//	*acs = AllocationChallenges(*d)
//	acs.ChallengeMap = make(map[string]*AllocOpenChallenge)
//	for _, challenge := range acs.OpenChallenges {
//		acs.ChallengeMap[challenge.ID] = challenge
//	}
//
//	return v, nil
//}

func (acs *AllocationChallenges) addChallenge(challenge *StorageChallenge) {
	//if acs.ChallengeMap == nil {
	//	acs.ChallengeMap = make(map[string]*AllocOpenChallenge)
	//}
	//
	//if _, ok := acs.ChallengeMap[challenge.ID]; !ok {
	//	oc := &AllocOpenChallenge{
	//		ID:        challenge.ID,
	//		BlobberID: challenge.BlobberID,
	//		CreatedAt: challenge.Created,
	//	}
	acs.OpenChallenges = append(acs.OpenChallenges, &AllocOpenChallenge{
		BlobberIndex: challenge.BlobberIndex,
		CreatedAt:    int64(challenge.Created),
		ChallengeID:  challenge.ID,
	})
	//	acs.ChallengeMap[challenge.ID] = oc
	//	return true
	//}
	//
	//return false
}

// Save saves the AllocationChallenges to MPT state
func (acs *AllocationChallenges) Save(state cstate.StateContextI, scAddress string) error {
	_, err := state.InsertTrieNode(acs.GetKey(scAddress), acs)
	return err
}

func (acs *AllocationChallenges) removeChallenge(challenge *StorageChallenge) bool {
	//if _, ok := acs.ChallengeMap[challenge.ID]; !ok {
	//	return false
	//}

	//delete(acs.ChallengeMap, challenge.ID)
	for i := range acs.OpenChallenges {
		if acs.OpenChallenges[i].ChallengeID == challenge.ID {
			acs.OpenChallenges = append(
				acs.OpenChallenges[:i], acs.OpenChallenges[i+1:]...)
			return true
		}
	}

	return false
}

//type allocationChallengesDecoder AllocationChallenges
