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

type DKGs []SimpleDKG

func newDKGs(t, n int) DKGs {

	if t > n {
		fmt.Println("Check the values of t and n")
		return nil
	}
	if t == 1 {
		fmt.Println("Library does not support t = 1, err mclBn_FrEvaluatePolynomial. Check the value of t")
		return nil
	}
	dkgs := make([]SimpleDKG, n)
	for i := range dkgs {
		dkgs[i] = MakeSimpleDKG(t, n)
		dkgs[i].ID = ComputeIDdkg(i)
		for j := range dkgs {
			dkgs[j].ID = ComputeIDdkg(j)
			dkgs[i].ComputeDKGKeyShare(dkgs[j].ID)
		}
	}
	return dkgs
}

/*TestMakeSimpleDKG - Test to check whether SecKey - secret key, mSec - master secret key and mVec - verification key are set
Here mSec[0] is the secretKey */
func TestMakeSimpleDKG(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		if dkg.mSec == nil {
			test.Errorf("The master secret key not set")
		}
		if dkg.Vvec == nil {
			test.Errorf("The verification key not set")
		}

	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/*TestMakeSimpleMultipleDKGs - Test to check creation of multiple DKGs works */
func TestMakeSimpleMultipleDKGs(test *testing.T) {

	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		for i := 0; i < n; i++ {
			if dkgs[i].mSec == nil {
				test.Errorf("The master secret key not set")
			}
			if dkgs[i].Vvec == nil {
				test.Errorf("The verification key not set")
			}
			if dkgs[i].secSharesMap == nil {
				test.Errorf("For PartyID %s The secShares not set %v", dkgs[i].ID.GetDecString(), dkgs[i].secSharesMap)
			}
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/*TestDkgSecretKey - Tests to check whether the polynomial substn method derives dkg shares from the secret of a party during the DKG process*/
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

/* testDKGProcess - Test for the DKG flow */
func testDKGProcess(t int, n int, test *testing.T) {

	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
		assert.NotNil(test, dkgs[i].SecKeyShareGroup)
	}
}

/* TestDKGProcess - calls the test, testDKGProcess*/
func TestDKGProcess(test *testing.T) { testDKGProcess(2, 3, test) }

/*aggDkg - Aggregate the DKG shares to form the GroupPrivateKeyShare */
func aggDkg(i int, dkgs DKGs, n int) {

	for j := 0; j < n; j++ {
		dkgs[i].receivedSecShares[j] = dkgs[j].secSharesMap[dkgs[i].ID]
	}

	if len(dkgs[i].receivedSecShares) == n {
		dkgs[i].AggregateShares()
	}

}

/* testRecoverGrpSignature - The test used to check the grp signature produced is the same for all miners*/
/* In the test, DKG is run for n miners to get the GroupPrivateKeyShare(GPKS). A party signs with
GPKS to form the Grp sign share. The array allPartyIDs is to keep track of party IDs to get it's unique k number of sig shares randomly for Recover GpSign*/

func testRecoverGrpSignature(t int, n int, test *testing.T) {
	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

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

			sigShare := bs.SignMsg()
			grpSignShareVerified := bs.VerifyGroupSignShare(sigShare)
			assert.True(test, grpSignShareVerified)
			partyMap[dkgs[i].ID] = sigShare

		}
		fmt.Printf("round %v) Group Sign share asserted to be true for all miners\n", rNumber)

		for i := 0; i < n; i++ {
			bs := MakeSimpleBLS(&dkgs[i])
			allPartyIDs := selectAllPartyIDs(dkgs, n)
			threshParty, threshSigs := calcRbo(allPartyIDs, t, partyMap)
			bs.RecoverGroupSig(threshParty, threshSigs)

			gpSign = bs.GpSign.GetHexString()
			fmt.Printf("round %v) The Group Signature : %v for the miner %v\n", rNumber, gpSign, bs.ID.GetDecString())
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

/* testVerifyGrpSignShares - Test to verify the Grp Signature Share which a miner computes by signing a message with GroupPrivateKeyShare(GPKS) is valid*/
func testVerifyGrpSignShares(t int, n int, test *testing.T) {
	dkgs := newDKGs(t, n)

	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

	for i := 0; i < n; i++ {

		bs := MakeSimpleBLS(&dkgs[i])
		bs.Msg = "VerifyGrpSignShare" + strconv.Itoa(i)
		sigShare := bs.SignMsg()
		grpSignShareVerified := bs.VerifyGroupSignShare(sigShare)

		if !grpSignShareVerified {
			test.Errorf("The grp signature share %v is not valid, which is computed by the party %v\n", sigShare.GetHexString(), bs.ID.GetDecString())
		}

	}
}

/* TestVerifyGrpSignShares - The test calls testVerifyGrpSignShares(t, n, test) which has the test for Grp Sign Share*/
func TestVerifyGrpSignShares(test *testing.T) { testVerifyGrpSignShares(2, 3, test) }

/* testVerifyWrongGrpSignShares - Test to verify whether an invalid Grp Signature share verifies to false */
func testVerifyWrongGrpSignShares(t int, n int, test *testing.T) {
	dkgs := newDKGs(t, n)

	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

	var wrongSigShare Sign

	for i := 0; i < n; i++ {

		bs := MakeSimpleBLS(&dkgs[i])
		bs.Msg = "VerifyGrpSignShare" + strconv.Itoa(i)
		wrongSigShare.SetHexString("BOGUSVALUE")
		grpSignShareVerified := bs.VerifyGroupSignShare(wrongSigShare)

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
	idVec := make([]PartyID, n)
	secVec := make([]Key, n)
	signVec := make([]Sign, n)
	for i := 0; i < n; i++ {
		err := idVec[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			b.Error(err)
		}
		err = secVec[i].Set(msk, &idVec[i])
		if err != nil {
			b.Error(err)
		}
		signVec[i] = *secVec[i].Sign("test message")
	}

	// recover signature
	var sig Sign
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		err := sig.Recover(signVec, idVec)
		if err != nil {
			b.Error(err)
		}
	}
}

/* BenchmarkRecoverSignature - Calls the function benchmarkRecoverSignature */
func BenchmarkRecoverSignature(b *testing.B) { benchmarkRecoverSignature(200, b) }
