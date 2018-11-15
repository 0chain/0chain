package bls

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/pmer/gobls"
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

/*TestRecoverSecretKey - Tests to check the secret key of a miner during the DKG process can be recovered back. Checking for poly sub method.*/
func TestRecoverSecretKey(test *testing.T) {

	variation := func(t, n int) {

		dkgs := newDKGs(t, n)
		secSharesFromMap := make([]Key, n)
		partyIdsFromMap := make([]PartyID, n)

		for i := 0; i < n; i++ {
			j := 0
			hmap := dkgs[i].secSharesMap
			for k, v := range hmap {
				secSharesFromMap[j] = v
				partyIdsFromMap[j] = k
				j++
			}

			if secSharesFromMap == nil {
				test.Errorf("Shares are not derived correctly %v\n", secSharesFromMap)
			}
			if partyIdsFromMap == nil {
				test.Errorf("PartyIds are not derived correctly %v\n", partyIdsFromMap)
			}

			var sec2 Key
			err := sec2.Recover(secSharesFromMap, partyIdsFromMap)
			if err != nil {
				test.Errorf("Recover shares Lagrange Interpolation error secSharesFromMap : %v\n ,Recover sec : %s\n, forIDs : %v\n", secSharesFromMap, sec2.GetHexString(), partyIdsFromMap)
				test.Error(err)
			}
			if !dkgs[i].mSec[0].IsEqual(&sec2) {
				test.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", dkgs[i].mSec[0].GetHexString(), sec2.GetHexString())
			}

		}
	}
	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* testDKGProcess - Test for the DKG flow */
func testDKGProcess(t int, n int, test *testing.T) {

	start := time.Now()
	dkgs := newDKGs(t, n)
	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}
	fmt.Printf("Time for the DKG process: %v for n : %v\n", time.Since(start), n)
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
GPKS to form the Bls sign share. The array selectRandIDs is to keep track of party IDs to get k number of sig shares randomly for Recover GpSign*/

func testRecoverGrpSignature(t int, n int, test *testing.T) {
	start := time.Now()

	dkgs := newDKGs(t, n)

	for i := 0; i < n; i++ {
		aggDkg(i, dkgs, n)
	}

	selectRandIDs := make([]PartyID, 0)
	var rbOutput string
	var prevRBO = "noRBO"
	sigSharesID := make(map[PartyID]Sign, n)

	for i := 0; i < n; i++ {

		bs := MakeSimpleBLS(&dkgs[i])
		bs.Msg = "RBO Tests"
		sigShare := bs.SignMsg()
		sigSharesID[dkgs[i].ID] = sigShare

	}
	for i := 0; i < n; i++ {
		bs := MakeSimpleBLS(&dkgs[i])
		selectRandIDs = addToSelectRandIDs(dkgs, n)
		threshParty, threshSigs := calcRbo(selectRandIDs, t, sigSharesID)
		bs.RecoverGroupSig(threshParty, threshSigs)

		rbOutput = bs.GpSign.GetHexString()
		if prevRBO != "noRBO" && prevRBO != rbOutput {
			test.Errorf("The rbOutput %v in %v is different for miner\n", i, rbOutput)
		}
		prevRBO = rbOutput
	}
	fmt.Printf("Time for the RecoverGrpSignature: %v, for t : %v and n : %v\n", time.Since(start), t, n)

}

/* TestRecGrpSign - The test calls testRecoverGrpSignature(t, n, test) which has the test for Gp Sign*/
func TestRecGrpSign(test *testing.T) { testRecoverGrpSignature(2, 3, test) }

/*calcRbo - To calculate the Gp Sign with any k number of Party IDs and its Bls signature share*/
func calcRbo(selectRandIDs []PartyID, t int, sigSharesID map[PartyID]Sign) (thresholdPartyIDs []PartyID, thresholdBlsSig []Sign) {

	thresholdCount := 1
	thresholdBlsSig = make([]Sign, 0)
	thresholdPartyIDs = make([]PartyID, 0)

	for thresholdCount <= t {

		randIndex := rand.Intn(len(selectRandIDs))

		thresholdPartyIDs = append(thresholdPartyIDs, selectRandIDs[randIndex])
		thresholdBlsSig = append(thresholdBlsSig, sigSharesID[selectRandIDs[randIndex]])

		selectRandIDs[randIndex] = selectRandIDs[len(selectRandIDs)-1]
		selectRandIDs = selectRandIDs[:len(selectRandIDs)-1]
		thresholdCount++

	}
	return thresholdPartyIDs, thresholdBlsSig
}

/*addToSelectRandIDs - The array selectRandIDs to select k number party IDs randomly */
func addToSelectRandIDs(dkgs DKGs, n int) []PartyID {
	selectRandIDs := make([]PartyID, 0)
	for j := 0; j < n; j++ {
		selectRandIDs = append(selectRandIDs, dkgs[j].ID)
	}
	return selectRandIDs
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
			test.Errorf("The bogus grp signature share %v cannot valid, which is computed by the party %v\n", wrongSigShare.GetHexString(), bs.ID.GetDecString())
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
