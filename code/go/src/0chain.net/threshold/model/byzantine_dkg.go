package model

type Attack struct {
	attacker PartyId
	attackee PartyId
}

type Defend struct {
	defender PartyId
	defendee PartyId
	keyShare KeyShare
}

type ByzantineDKG struct {
	Simple SimpleDKG

	// TODO: Should these include my attacks and defends?
	seenAttacks []bool
	seenDefends []KeyShare

	disqualified []bool
}

func NewByzantineDKG(t, n int) ByzantineDKG {
	return ByzantineDKG{
		Simple:       NewSimpleDKG(t, n),
		seenAttacks:  make([]bool, n*n),
		seenDefends:  make([]KeyShare, n*n),
		disqualified: make([]bool, 4),
	}
}

func (d *ByzantineDKG) attack(a Attack) *bool {
	return &d.seenAttacks[int(a.attackee)*d.Simple.N+int(a.attacker)]
}

func (d *ByzantineDKG) defend(def Defend) *KeyShare {
	return &d.seenDefends[int(def.defendee)*d.Simple.N+int(def.defender)]
}

// WARNING: This function is extremely dangerous because a malicious player
//          could goad you into trying to attack them. If this happened, the
//          honest nodes would disqualify you.
func (d *ByzantineDKG) AttackUnreceived() []Attack {
	attacks := make([]Attack, d.Simple.N)

	i := PartyId(1)
	n := PartyId(d.Simple.N)
	for ; i < n; i++ {
		if d.Simple.GetShareFrom(i) == EmptyKeyShare {
			attacks = append(attacks, Attack{
				attacker: 0,
				attackee: i,
			})
		}
	}

	return attacks
}

// We can also override SimpleDKG.ReceiveShare() to return an Attack on the
// sender instead of an error if their share didn't validate. However, this is
// also dangerous because, once again, a malicious party may manipulate us into
// doing that when they are not actually failing, which would, at the end, cause
// honest nodes to disqualify us.
//
//func (d *ByzantineDKG) ReceiveShare(i PartyId, share KeyShare) *Attack {
//    ...
//}

func (d *ByzantineDKG) ReceiveAttack(a Attack) ([]PartyId, []Defend) {
	var disqualifications []PartyId = nil
	var defends []Defend = nil

	if *d.attack(a) {
		// We have already seen this attack. It's a repeat.
		return disqualifications, defends
	}

	*d.attack(a) = true

	// TODO: Determine if this disqualifies the attackee.
	//disqualifications := make([]PartyId, 0)

	// TODO: Determine if we should create Defend actions or a disqualification.
	//defends := make([]Defend, 0)

	return disqualifications, defends
}

func (d *ByzantineDKG) ReceiveDefend(def Defend) []PartyId {
	var disqualifications []PartyId = nil

	if *d.defend(def) != EmptyKeyShare {
		// We have already seen this defend. It's a repeat.
		return disqualifications
	}

	*d.defend(def) = def.keyShare

	// TODO: Determine if this disqualifies the attackers.
	//disqualifications := make([]PartyId, 0)

	return disqualifications
}
