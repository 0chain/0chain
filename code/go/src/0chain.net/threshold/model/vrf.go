// Threshold-based Verifiable Random Function
//
// Produces random values by:
//   random_n = SHA3(GroupSign(roundNum || random_{n-1}))
//
// Everybody in the group will arrive at the same value, and nobody will know
// what the next value will be before their group signature is done.

// Our group signature is the proof component of the VRF.
//
// Hashing the proof gives us the random output.
//
// The VRF produces values that appear pseudorandom to all 3rd party observers
// without the proof, yet which appear deterministic once the proof is seen.
//
// Revealing the proof to anyone allows them to validate our random
// value. Anyone can do this by the following algorithm:
//   1) Validate the proof is the signature of the concatenation of the round
//      number and the previous random output, and
//   2) Validate the random output is the hash of the proof.

package model

import (
	"bytes"
	"crypto"
	"encoding/binary"
)

type Round uint64
type RandomOutput [4]uint64

var EmptyRandomOutput = RandomOutput{}

type VRF struct {
	builder SignatureBuilder
}

func NewVRF(p *Party, round Round, prev RandomOutput) VRF {
	var msg bytes.Buffer
	binary.Write(&msg, binary.LittleEndian, round)
	binary.Write(&msg, binary.LittleEndian, prev)

	return VRF{
		builder: NewSignatureBuilder(p, msg.Bytes()),
	}
}

func (v *VRF) ReceiveShare(i PartyId, share SignatureShare) error {
	return v.builder.receiveShare(i, share)
}

func (v *VRF) Output() RandomOutput {
	if !v.builder.isDone() {
		return EmptyRandomOutput
	}

	var proof GroupSignature
	var proofBytes bytes.Buffer

	proof = v.builder.GroupSignature
	binary.Write(&proofBytes, binary.LittleEndian, proof)

	state := crypto.SHA3_256.New()
	state.Write(proofBytes.Bytes())

	var output RandomOutput
	var outputBytes bytes.Buffer

	state.Sum(outputBytes.Bytes())
	binary.Read(&outputBytes, binary.LittleEndian, output)

	// FIXME: Even though this is a VRF, we don't save the proof for later
	// publication?

	return output
}
