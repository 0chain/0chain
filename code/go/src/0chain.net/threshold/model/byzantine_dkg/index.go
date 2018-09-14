package model_byzantine_dkg

import (
	"0chain.net/threshold/model"
	simple "0chain.net/threshold/model/simple_dkg"
)

type Attack struct {
	attacker model.PartyId
	attackee model.PartyId
}

type Defend struct {
	defender model.PartyId
	defendee model.PartyId
	keyShare simple.KeyShare
}

type DKG struct {
	Simple simple.DKG

	// TODO: Should these include my attacks and defends?
	seenAttacks []bool
	seenDefends []simple.KeyShare

	disqualified []bool
}

func New(t, n int) DKG {
	return DKG{
		Simple:       simple.New(t, n),
		seenAttacks:  make([]bool, n*n),
		seenDefends:  make([]simple.KeyShare, n*n),
		disqualified: make([]bool, 4),
	}
}

func (d *DKG) attack(a Attack) *bool {
	return &d.seenAttacks[int(a.attackee)*d.Simple.N+int(a.attacker)]
}

func (d *DKG) defend(def Defend) *simple.KeyShare {
	return &d.seenDefends[int(def.defendee)*d.Simple.N+int(def.defender)]
}

// WARNING: This function is extremely dangerous because a malicious player
//          could goad you into trying to attack them. If this happened, the
//          honest nodes would disqualify you.
func (d *DKG) AttackUnreceived() []Attack {
	attacks := make([]Attack, d.Simple.N)

	i := model.PartyId(1)
	n := model.PartyId(d.Simple.N)
	for ; i < n; i++ {
		if d.Simple.GetShareFrom(i) == simple.EmptyKeyShare {
			attacks = append(attacks, Attack{
				attacker: 0,
				attackee: i,
			})
		}
	}

	return attacks
}

// We can also override simple.ReceiveShare() to return an Attack on the sender
// instead of an error if their share didn't validate. However, this is also
// dangerous because, once again, a malicious party may manipulate us into doing
// that when they are not actually failing, which would, at the end, cause
// honest nodes to disqualify us.
//
//func (d *DKG) ReceiveShare(i model.PartyId, share KeyShare) *Attack {
//    ...
//}

func (d *DKG) ReceiveAttack(a Attack) ([]model.PartyId, []Defend) {
	var disqualifications []model.PartyId = nil
	var defends []Defend = nil

	if *d.attack(a) {
		// We have already seen this attack. It's a repeat.
		return disqualifications, defends
	}

	*d.attack(a) = true

	// TODO: Determine if this disqualifies the attackee.
	//disqualifications := make([]model.PartyId, 0)

	// TODO: Determine if we should create Defend actions or a disqualification.
	//defends := make([]Defend, 0)

	return disqualifications, defends
}

func (d *DKG) ReceiveDefend(def Defend) []model.PartyId {
	var disqualifications []model.PartyId = nil

	if *d.defend(def) != simple.EmptyKeyShare {
		// We have already seen this defend. It's a repeat.
		return disqualifications
	}

	*d.defend(def) = def.keyShare

	// TODO: Determine if this disqualifies the attackers.
	//disqualifications := make([]model.PartyId, 0)

	return disqualifications
}
