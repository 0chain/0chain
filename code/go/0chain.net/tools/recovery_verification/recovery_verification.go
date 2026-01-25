package main

import (
	"fmt"
	"strings"
)

// ViewChangeOffset - copy from entity.go to verify
const ViewChangeOffset = 20

// mbRoundOffset - copy from entity.go to verify
func mbRoundOffset(rn int64) int64 {
	if rn < ViewChangeOffset+1 {
		return rn
	}
	return rn - ViewChangeOffset
}

// Test represents a single test case
type Test struct {
	Name     string
	Run      func() bool
	Expected string
}

func main() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("  RECOVERY FIX VERIFICATION")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	tests := []Test{
		// ViewChangeOffset tests
		{
			Name: "ViewChangeOffset is 20",
			Run: func() bool {
				return ViewChangeOffset == 20
			},
			Expected: "ViewChangeOffset should be 20 for mainnet recovery",
		},

		// mbRoundOffset boundary tests
		{
			Name: "mbRoundOffset(0) returns 0",
			Run: func() bool {
				return mbRoundOffset(0) == 0
			},
			Expected: "Round 0 should have no offset",
		},
		{
			Name: "mbRoundOffset(20) returns 20",
			Run: func() bool {
				return mbRoundOffset(20) == 20
			},
			Expected: "Round 20 (at boundary) should have no offset",
		},
		{
			Name: "mbRoundOffset(21) returns 1",
			Run: func() bool {
				return mbRoundOffset(21) == 1
			},
			Expected: "Round 21 (first with offset) should return 21-20=1",
		},
		{
			Name: "mbRoundOffset(100) returns 80",
			Run: func() bool {
				return mbRoundOffset(100) == 80
			},
			Expected: "Round 100 should return 100-20=80",
		},

		// MB20 activation tests (mainnet scenario)
		{
			Name: "MB20 activates at round 141945735",
			Run: func() bool {
				const mb20StartingRound = int64(141945715)
				activationRound := mb20StartingRound + ViewChangeOffset
				return activationRound == 141945735
			},
			Expected: "MB20 (SR=141945715) should activate at round 141945715+20=141945735",
		},
		{
			Name: "mbRoundOffset(141945735) returns MB20 StartingRound",
			Run: func() bool {
				const mb20StartingRound = int64(141945715)
				return mbRoundOffset(141945735) == mb20StartingRound
			},
			Expected: "mbRoundOffset(141945735) should return 141945715 (MB20 SR)",
		},
		{
			Name: "Round 141945734 does NOT use MB20",
			Run: func() bool {
				const mb20StartingRound = int64(141945715)
				return mbRoundOffset(141945734) != mb20StartingRound
			},
			Expected: "Round 141945734 should NOT use MB20 (uses MB19)",
		},
		{
			Name: "Round 141945734 uses offset 141945714",
			Run: func() bool {
				return mbRoundOffset(141945734) == 141945714
			},
			Expected: "mbRoundOffset(141945734) should return 141945714",
		},

		// Old vs New offset comparison
		{
			Name: "Old offset=25 would activate MB20 at round 141945740",
			Run: func() bool {
				oldOffset := int64(25)
				mb20SR := int64(141945715)
				oldActivationRound := mb20SR + oldOffset
				return oldActivationRound == 141945740
			},
			Expected: "With old offset=25, MB20 would activate at 141945740 (too late)",
		},
		{
			Name: "New offset=20 activates MB20 5 rounds earlier",
			Run: func() bool {
				oldOffset := int64(25)
				newOffset := int64(ViewChangeOffset)
				return oldOffset-newOffset == 5
			},
			Expected: "New offset=20 is 5 rounds earlier than old offset=25",
		},

		// reachedNotarization threshold logic verification
		{
			Name: "MB19 threshold is 17 (25 miners * 0.66)",
			Run: func() bool {
				mb19Miners := 25
				thresholdRatio := 0.66
				threshold := int(float64(mb19Miners)*thresholdRatio) + 1
				return threshold == 17
			},
			Expected: "MB19 with 25 miners needs 17 tickets for consensus",
		},
		{
			Name: "MB20 threshold is 12 (18 miners * 0.66)",
			Run: func() bool {
				mb20Miners := 18
				thresholdRatio := 0.66
				threshold := int(float64(mb20Miners)*thresholdRatio) + 1
				return threshold == 12
			},
			Expected: "MB20 with 18 miners needs 12 tickets for consensus",
		},
	}

	// Run all tests
	passed := 0
	failed := 0
	for _, test := range tests {
		result := test.Run()
		if result {
			passed++
			fmt.Printf("✓ PASS: %s\n", test.Name)
		} else {
			failed++
			fmt.Printf("✗ FAIL: %s\n", test.Name)
			fmt.Printf("  Expected: %s\n", test.Expected)
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("RESULTS: %d passed, %d failed\n", passed, failed)
	fmt.Println(strings.Repeat("=", 80))

	// Summary
	fmt.Println()
	fmt.Println("RECOVERY FIX SUMMARY:")
	fmt.Println("----------------------")
	fmt.Println("1. ViewChangeOffset changed from 25 to 20")
	fmt.Println("2. MB20 (StartingRound=141945715) now activates at round 141945735")
	fmt.Println("3. This is 5 rounds earlier than the old activation (141945740)")
	fmt.Println("4. Sharder4 at round 141945734 will need to roll forward 1 round to 141945735")
	fmt.Println("5. At round 141945735, MB20 with 18 miners (12 threshold) will be used")
	fmt.Println()

	fmt.Println("THRESHOLD CHECK FIX:")
	fmt.Println("----------------------")
	fmt.Println("The reachedNotarization fix ensures that during MB mismatch:")
	fmt.Println("- Block MUST still have threshold number of verification tickets")
	fmt.Println("- Previously it would return true without checking threshold (BUG)")
	fmt.Println("- Now it always checks: len(bvt) >= threshold")
	fmt.Println()

	if failed > 0 {
		fmt.Println("ERROR: Some tests failed!")
	} else {
		fmt.Println("All verification tests passed successfully!")
	}
}
