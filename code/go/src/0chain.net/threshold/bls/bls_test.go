package bls

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"0chain.net/encryption"
	"github.com/pmer/gobls"
	"github.com/stretchr/testify/assert"
)

const CurveFp254BNb = 0

type DKGs []DKG

/*VerificationKey - Is of type gobls.PublicKey*/
type VerificationKey = gobls.PublicKey

func newDKGs(t, n int) DKGs {

	if t > n {
		fmt.Println("Check the values of t and n")
		return nil
	}
	if t == 1 {
		fmt.Println("Library does not support t = 1, err mclBn_FrEvaluatePolynomial. Check the value of t")
		return nil
	}
	dkgs := make([]DKG, n)
	for i := range dkgs {
		dkgs[i] = MakeDKG(t, n)
		dkgs[i].ID = ComputeIDdkg(i)
		for j := range dkgs {
			dkgs[j].ID = ComputeIDdkg(j)
			dkgs[i].ComputeDKGKeyShare(dkgs[j].ID)
		}
	}
	return dkgs
}

/*TestMakeMultipleDKGs - Test to check creation of multiple BLS-DKGs works */
func TestMakeMultipleDKGs(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		for i := 0; i < n; i++ {

			assert.NotNil(test, dkgs[i].mSec)
			if dkgs[i].mSec == nil {
				test.Errorf("The master secret key not set")
			}

			assert.NotNil(test, dkgs[i].secSharesMap)
			if dkgs[i].secSharesMap == nil {
				test.Errorf("For PartyID %s The secShares not set %v", dkgs[i].ID.GetDecString(), dkgs[i].secSharesMap)
			}
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/*TestDkgSecretKey - Tests to check whether the polynomial substn method derives GSKSS from the secret of a party*/
func TestDkgSecretKey(test *testing.T) {

	variation := func(t, n int) {

		dkgs := newDKGs(t, n)
		allPartyIDs := make([]PartyID, 0)
		dkgSharesCalcByParty := make([]Key, 0)

		for i := 0; i < n; i++ {
			allPartyIDs = append(allPartyIDs, dkgs[i].ID)
		}

		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				dkgSharesCalcByParty = append(dkgSharesCalcByParty, dkgs[i].secSharesMap[dkgs[j].ID])
			}

			assert.NotNil(test, allPartyIDs)
			assert.NotNil(test, dkgSharesCalcByParty)

			var sec2 Key
			err := sec2.Recover(dkgSharesCalcByParty, allPartyIDs)
			if err != nil {
				test.Errorf("Recover shares Lagrange Interpolation error for party")
				test.Error(err)
			}
			if !dkgs[i].mSec[0].IsEqual(&sec2) {
				test.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", dkgs[i].mSec[0].GetHexString(), sec2.GetHexString())
			}
			dkgSharesCalcByParty = dkgSharesCalcByParty[:0]

		}
	}
	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/*testVerifyGSKSS - Tests to verify the received GSKSS from a party. */
/*Consider two parties Mj and Mi.
  Mj sends Mi's GSKSS which Mj calculated to Mi and Mi verifies it by calling polynomial sub method with Mj's Vvec and Mi's ID which gives computedPubFromVvec
  The public key on the received GSKSS from Mj should be equal to computedPubFromVvec
*/
func testVerifyGSKSS(t int, n int, test *testing.T) {
	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
		assert.NotNil(test, dkgs[i].SecKeyShareGroup)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {

			Vvec := gobls.GetMasterPublicKey(dkgs[j].mSec)
			computedPubFromVvec, err := computeGpPublicKeyShareShares(Vvec, dkgs[i].ID)
			var recGSKSS Key

			if err == nil {
				recGSKSS = dkgs[j].secSharesMap[dkgs[i].ID]
			}
			assert.True(test, computedPubFromVvec.IsEqual(recGSKSS.GetPublicKey()))

			if !computedPubFromVvec.IsEqual(recGSKSS.GetPublicKey()) {
				test.Errorf("Mismatch in received GSKSS with vVec, so computedPubFromVvec is %v and recGSKSS pub is %v\n", computedPubFromVvec.GetHexString(), recGSKSS.GetPublicKey().GetHexString())
			}
		}

	}

}

/*TestVerifyGSKSS - Tests to verify the received GSKSS from a party*/
func TestVerifyGSKSS(test *testing.T) { testVerifyGSKSS(2, 3, test) }

/*computeGpPublicKeyShareShares - Derive the correpndg pubKey of the received GSKSS through polynomial substitution method with Vvec of sender and ID of receiver*/
func computeGpPublicKeyShareShares(recVvec []VerificationKey, fromID PartyID) (VerificationKey, error) {

	var pubKey VerificationKey
	err := pubKey.Set(recVvec, &fromID)
	if err != nil {
		return VerificationKey{}, nil
	}
	return pubKey, nil
}

/*aggDkg - Aggregate the received GSKSS to form the GroupSecretKeyShare (GSKS) */
func aggDkg(i int, dkgs DKGs, n int) {
	recGSKSS := make([]Key, n)

	for j := 0; j < n; j++ {
		recGSKSS[j] = dkgs[j].secSharesMap[dkgs[i].ID]
	}

	if len(recGSKSS) == n {
		computedGSKS := aggRecGSKSS(recGSKSS)
		dkgs[i].SecKeyShareGroup = computedGSKS
	}

}

/*aggRecGSKSS - Each party aggregates the received GSKSS */
func aggRecGSKSS(recGSKSS []Key) Key {
	var sec Key

	for i := 0; i < len(recGSKSS); i++ {
		sec.Add(&recGSKSS[i])
	}
	return sec
}

/*testDKGGpPublicKey - Tests for the grp public key*/
/* In the test, BLS-DKG process is done and the committed verification vectors are added to compute the GroupsVvec
   GroupsVvec[0] is the groupPublicKey */

func testDkgGpPublicKey(t int, n int, test *testing.T) {
	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

	Vvecs := make([][]VerificationKey, n)

	for i := range Vvecs {

		Vvec := gobls.GetMasterPublicKey(dkgs[i].mSec)
		assert.NotNil(test, Vvec)

		Vvecs[i] = make([]VerificationKey, t)
		eachVvec := make([]VerificationKey, t)
		for j := range Vvecs[i] {

			eachVvec[j] = Vvec[j]
			Vvecs[i][j] = eachVvec[j]

		}

	}
	groupsVvec := calcGroupsVvec(Vvecs, t, n)
	assert.NotNil(test, groupsVvec)
	assert.True(test, (len(groupsVvec) == t))

	groupPublicKey := groupsVvec[0]
	assert.NotNil(test, groupPublicKey)

}

/*TestDKGGpPublicKey - Tests for the grp public key*/
func TestDkgGpPublicKey(test *testing.T) { testDkgGpPublicKey(2, 3, test) }

/*calcGroupsVvec - Aggregates the committed verification vectors by all partys to get the Groups Vvec */
func calcGroupsVvec(Vvecs [][]VerificationKey, t int, n int) []VerificationKey {

	groupsVvec := make([]VerificationKey, t)

	for i := range Vvecs {

		for j := range Vvecs[i] {

			pub2 := Vvecs[i][j]
			pub1 := groupsVvec[j]
			pub1.Add(&pub2)
			groupsVvec[j] = pub1
		}
	}
	return groupsVvec
}

/* testRecoverGrpSignature - The test used to check the grp signature produced is the same for all miners*/
/* In the test, DKG is run for n miners to get the GroupPrivateKeyShare(GPKS). A party signs with
GPKS to form the Grp sign share. The array allPartyIDs is to keep track of party IDs to get it's unique k number of sig shares randomly for Recover GpSign*/

func testRecoverGrpSignature(t int, n int, test *testing.T) {

	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

	Vvecs := make([][]VerificationKey, n)

	for i := range Vvecs {

		Vvec := gobls.GetMasterPublicKey(dkgs[i].mSec)
		assert.NotNil(test, Vvec)

		Vvecs[i] = make([]VerificationKey, t)
		eachVvec := make([]VerificationKey, t)
		for j := range Vvecs[i] {

			eachVvec[j] = Vvec[j]
			Vvecs[i][j] = eachVvec[j]

		}

	}

	groupsVvec := calcGroupsVvec(Vvecs, t, n)
	assert.NotNil(test, groupsVvec)
	assert.True(test, (len(groupsVvec) == t))

	groupPublicKey := groupsVvec[0]
	assert.NotNil(test, groupPublicKey)

	var msg Message
	var rNumber int64 = 1
	var gpSign string
	var rbOutput string
	var prevRBO = encryption.Hash("0chain")
	var prevMinerRBO = "noRBO"

	//partyMap has the partyID and its corresponding Grp sign share
	partyMap := make(map[PartyID]Sign, n)

	for rNumber <= 1000 {
		fmt.Printf("*Starting round %v)\n", rNumber)
		for i := 0; i < n; i++ {

			bs := MakeSimpleBLS(&dkgs[i])
			bs.Msg = strconv.FormatInt(rNumber, 10) + prevRBO //msg = r || RBO(r-1), r -> round
			fmt.Printf("round %v) The message : %v signed by miner %v\n", rNumber, bs.Msg, bs.ID.GetDecString())
			msg = bs.Msg

			sigShare := bs.SignMsg()

			//verify the grp sign share computed by each miner
			grpSignShareVerified := verifyGroupSignShare(sigShare, bs.ID, groupsVvec, bs.Msg)
			assert.True(test, grpSignShareVerified)
			fmt.Printf("round %v) Group Sign share asserted to be true for all miners\n", rNumber)

			partyMap[dkgs[i].ID] = sigShare

		}

		for i := 0; i < n; i++ {
			bs := MakeSimpleBLS(&dkgs[i])
			allPartyIDs := selectAllPartyIDs(dkgs, n)
			threshParty, threshSigs := calcRbo(allPartyIDs, t, partyMap)
			bs.RecoverGroupSig(threshParty, threshSigs)

			gpSign = bs.GpSign.GetHexString()
			fmt.Printf("round %v) The Group Signature : %v for the miner %v\n", rNumber, gpSign, bs.ID.GetDecString())

			//verify the grp sign with the groupPublicKey
			grpSignVerified := verifyGroupSign(bs.GpSign, groupPublicKey, msg)
			assert.True(test, grpSignVerified)
			fmt.Printf("round %v) Group Sign asserted to be true for all miners\n", rNumber)

			rbOutput = encryption.Hash(gpSign)
			fmt.Printf("round %v) The rbOutput : %v for the miner %v\n", rNumber, rbOutput, bs.ID.GetDecString())

			if prevMinerRBO != "noRBO" && prevMinerRBO != rbOutput {
				assert.Equal(test, prevMinerRBO, rbOutput)
				test.Errorf("round %v) The rbOutput %v is different for miner %v\n", rNumber, rbOutput, bs.ID.GetDecString())
			}
			prevMinerRBO = rbOutput
		}
		fmt.Printf("round %v) The RBO %v is same for all miners for the round %v\n", rNumber, rbOutput, rNumber)
		prevRBO = rbOutput
		prevMinerRBO = "noRBO"
		rNumber++
		fmt.Println("----------------------------------------------------------------------------------------------------------------------")
	}
}

/* TestRecGrpSign - The test calls testRecoverGrpSignature(t, n, test) which has the test for Gp Sign*/
func TestRecGrpSign(test *testing.T) { testRecoverGrpSignature(2, 3, test) }

/*calcRbo - To calculate the Gp Sign with any k number of unique Party IDs and its Bls signature share*/
func calcRbo(allPartyIDs []PartyID, t int, partyMap map[PartyID]Sign) (threshPartys []PartyID, threshSigs []Sign) {

	var thresholdCount = 1
	threshSigs = make([]Sign, 0)
	threshPartys = make([]PartyID, 0)

	for thresholdCount <= t {

		randIndex := rand.Intn(len(allPartyIDs))

		//selecting a party's Grp Sign share randomly
		threshPartys = append(threshPartys, allPartyIDs[randIndex])
		threshSigs = append(threshSigs, partyMap[allPartyIDs[randIndex]])

		//removing that party's ID from the array allPartyIDs, to avoid duplicates to compute Grp Sign
		allPartyIDs[randIndex] = allPartyIDs[len(allPartyIDs)-1]
		allPartyIDs = allPartyIDs[:len(allPartyIDs)-1]

		thresholdCount++

	}
	return threshPartys, threshSigs
}

/*selectAllPartyIDs - The array allPartyIDs has all the party IDs. This array will be used later on to select k random sig shares to derive grp sign */
func selectAllPartyIDs(dkgs DKGs, n int) []PartyID {
	allPartyIDs := make([]PartyID, 0)
	for j := 0; j < n; j++ {
		allPartyIDs = append(allPartyIDs, dkgs[j].ID)
	}
	return allPartyIDs
}

/* testVerifyGrpSignShares - Test to verify the Grp Signature Share which a miner computes (GSS) by signing a message with GSKS is valid*/
func testVerifyGrpSignShares(t int, n int, test *testing.T) {

	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

	Vvecs := make([][]VerificationKey, n)

	for i := range Vvecs {

		Vvec := gobls.GetMasterPublicKey(dkgs[i].mSec)
		assert.NotNil(test, Vvec)

		Vvecs[i] = make([]VerificationKey, t)
		eachVvec := make([]VerificationKey, t)
		for j := range Vvecs[i] {

			eachVvec[j] = Vvec[j]
			Vvecs[i][j] = eachVvec[j]

		}

	}
	groupsVvec := calcGroupsVvec(Vvecs, t, n)
	assert.NotNil(test, groupsVvec)
	assert.True(test, (len(groupsVvec) == t))

	groupPublicKey := groupsVvec[0]
	assert.NotNil(test, groupPublicKey)

	for i := 0; i < n; i++ {

		bs := MakeSimpleBLS(&dkgs[i])
		bs.Msg = "VerifyGrpSignShare" + strconv.Itoa(i)
		sigShare := bs.SignMsg()
		grpSignShareVerified := verifyGroupSignShare(sigShare, bs.ID, groupsVvec, bs.Msg)

		assert.True(test, grpSignShareVerified)

		//optional double - check
		assert.True(test, sigShare.Verify(&(*bs.SecKeyShareGroup.GetPublicKey()), bs.Msg))

		if !grpSignShareVerified {
			test.Errorf("The grp signature share %v is not valid, which is computed by the party %v\n", sigShare.GetHexString(), bs.ID.GetDecString())
		}
	}
}

/* TestVerifyGrpSignShares - The test calls testVerifyGrpSignShares(t, n, test) which has the test for Grp Sign Share*/
func TestVerifyGrpSignShares(test *testing.T) { testVerifyGrpSignShares(2, 3, test) }

/*verifyGroupSignShare - To verify the Gp sign share (GSS) */
/* GSS is verified by calling polynomial substitution method with the Groups Vvec and the party ID which computed it*/
func verifyGroupSignShare(grpSignShare Sign, fromID PartyID, groupsVvec []VerificationKey, msg Message) bool {

	var pubK VerificationKey
	err := pubK.Set(groupsVvec, &fromID)
	if err != nil {
		return false
	}

	if !grpSignShare.Verify(&pubK, msg) {
		return false
	}
	return true

}

/*verifyGroupSign - To verify the Gp sign (GS) */
/* GS is verified by the groupPublicKey */
func verifyGroupSign(grpSign Sign, groupPublicKey VerificationKey, msg Message) bool {

	if !grpSign.Verify(&groupPublicKey, msg) {
		return false
	}
	return true

}

/*testVerifyWrongGrpSignShares - Test to verify whether an invalid Grp Signature share verifies to false */

func testVerifyWrongGrpSignShares(t int, n int, test *testing.T) {

	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

	Vvecs := make([][]VerificationKey, n)

	for i := range Vvecs {

		Vvec := gobls.GetMasterPublicKey(dkgs[i].mSec)
		assert.NotNil(test, Vvec)

		Vvecs[i] = make([]VerificationKey, t)
		eachVvec := make([]VerificationKey, t)
		for j := range Vvecs[i] {

			eachVvec[j] = Vvec[j]
			Vvecs[i][j] = eachVvec[j]

		}

	}

	groupsVvec := calcGroupsVvec(Vvecs, t, n)
	assert.NotNil(test, groupsVvec)
	assert.True(test, (len(groupsVvec) == t))

	groupPublicKey := groupsVvec[0]
	assert.NotNil(test, groupPublicKey)

	var wrongSigShare Sign

	for i := 0; i < n; i++ {

		bs := MakeSimpleBLS(&dkgs[i])
		bs.Msg = "VerifyGrpSignShare" + strconv.Itoa(i)
		wrongSigShare.SetHexString("BOGUSVALUE")
		grpSignShareVerified := verifyGroupSignShare(wrongSigShare, bs.ID, groupsVvec, bs.Msg)
		assert.False(test, grpSignShareVerified)

		if grpSignShareVerified {
			test.Errorf("The bogus grp signature share %v cannot be valid, which is computed by the party %v\n", wrongSigShare.GetHexString(), bs.ID.GetDecString())
		}

	}
}

/* TestVerifyWrongGrpSignShares - The test calls testVerifyWrongGrpSignShares(t, n, test) which has the test for Grp Sign Share*/
func TestVerifyWrongGrpSignShares(test *testing.T) { testVerifyWrongGrpSignShares(2, 3, test) }

/* BenchmarkDeriveGpSignShare - Benchmark for deriving the Gp Sign Share*/
func BenchmarkDeriveGpSignShare(b *testing.B) {
	b.StopTimer()

	err := gobls.Init(gobls.CurveFp254BNb)
	if err != nil {
		b.Errorf("Curve not initialized")
	}
	var sec Key
	for n := 0; n < b.N; n++ {
		sec.SetByCSPRNG()
		b.StartTimer()
		m := "GpSignShareMsg" + strconv.Itoa(n)
		sec.Sign(m)
		b.StopTimer()

	}
}

/* BenchmarkVerifyGpSignShare - Benchmark for verifying the Gp Sign Share*/
func BenchmarkVerifyGpSignShare(b *testing.B) {
	b.StopTimer()
	err := gobls.Init(gobls.CurveFp254BNb)
	if err != nil {
		b.Errorf("Curve not initialized")
	}
	var sec Key
	for n := 0; n < b.N; n++ {
		sec.SetByCSPRNG()
		pub := sec.GetPublicKey()
		m := "GpSignShareVerifyMsg" + strconv.Itoa(n)
		sig := sec.Sign(m)
		b.StartTimer()
		sig.Verify(pub, m)
		b.StopTimer()
	}
}

/* benchmarkDeriveDkgShare - Benchmark for polynomial substitution method used in deriving the DKG shares for a party*/
func benchmarkDeriveDkgShare(t int, b *testing.B) {
	b.StopTimer()
	err := gobls.Init(gobls.CurveFp254BNb)
	if err != nil {
		b.Errorf("Curve not initialized")
	}
	var sec Key
	sec.SetByCSPRNG()
	msk := sec.GetMasterSecretKey(t)
	var forID PartyID
	for n := 0; n < b.N; n++ {
		err = forID.SetDecString(strconv.Itoa(n + 1))
		if err != nil {
			b.Error(err)
		}
		b.StartTimer()
		err := sec.Set(msk, &forID)
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
	}
}

/* BenchmarkDeriveDkgShare - calls the benchmark test benchmarkDeriveDkgShare which tests deriving the DKG shares for a party*/
func BenchmarkDeriveDkgShare(b *testing.B) { benchmarkDeriveDkgShare(1000, b) }

/* benchmarkRecoverSignature - Benchmark for Recover Grp Sign which is used to compute the Grp Signature */
func benchmarkRecoverSignature(k int, b *testing.B) {
	b.StopTimer()
	err := gobls.Init(gobls.CurveFp254BNb)
	if err != nil {
		b.Errorf("Curve not initialized")
	}
	var sec Key
	sec.SetByCSPRNG()
	msk := sec.GetMasterSecretKey(k)

	n := k
	ids := make([]PartyID, n)
	secKy := make([]Key, n)
	signs := make([]Sign, n)
	for i := 0; i < n; i++ {
		err = ids[i].SetDecString(strconv.Itoa(i + 1))
		if err != nil {
			b.Error(err)
		}
		err = secKy[i].Set(msk, &ids[i])
		if err != nil {
			b.Error(err)
		}
		signs[i] = *secKy[i].Sign("test message")
	}

	// recover from all k signatures
	var sig Sign
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		err := sig.Recover(signs, ids)
		if err != nil {
			b.Error(err)
		}
	}
}

/* BenchmarkRecoverSignature - Calls the function benchmarkRecoverSignature */
func BenchmarkRecoverSignature(b *testing.B) { benchmarkRecoverSignature(200, b) }
