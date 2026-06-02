package main

import (
	"fmt"
	"strings"
)

// CommitAnalysis categorizes and analyzes all commits on the branch
type CommitAnalysis struct {
	Hash         string
	Description  string
	Category     string
	Needed       string
	Reason       string
}

func main() {
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("  COMMIT ANALYSIS: fix/dkg-broadcast-fee branch vs staging")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println()

	commits := []CommitAnalysis{
		// ============= ESSENTIAL FOR RECOVERY =============
		{
			Hash:        "UNCOMMITTED",
			Description: "ViewChangeOffset = 20 (entity.go)",
			Category:    "ESSENTIAL",
			Needed:      "YES",
			Reason:      "Core recovery fix - shifts MB20 activation from round 141945740 to 141945735",
		},
		{
			Hash:        "UNCOMMITTED",
			Description: "reachedNotarization always checks threshold (protocol_block.go)",
			Category:    "ESSENTIAL",
			Needed:      "YES",
			Reason:      "Fixes bug where MB mismatch bypassed threshold check - was the root cause",
		},
		{
			Hash:        "8383f92a0",
			Description: "require verification tickets during MB mismatch",
			Category:    "ESSENTIAL",
			Needed:      "SUPERSEDED",
			Reason:      "Superseded by uncommitted fix which is more complete (always checks threshold)",
		},

		// ============= IMPORTANT FIXES =============
		{
			Hash:        "c229e44ce",
			Description: "move SetRandomSeed after verification",
			Category:    "IMPORTANT",
			Needed:      "YES",
			Reason:      "Prevents failed blocks from poisoning VRF state",
		},
		{
			Hash:        "7a97add24",
			Description: "allow restart of rounds in Complete phase",
			Category:    "IMPORTANT",
			Needed:      "YES",
			Reason:      "Fixes deadlock in rounds stuck at Complete phase without notarization",
		},
		{
			Hash:        "a3a2f3631",
			Description: "load previous MB's DKG on startup",
			Category:    "IMPORTANT",
			Needed:      "YES",
			Reason:      "Prevents 'DKG is nil' errors after restart near MB boundaries",
		},
		{
			Hash:        "28a53c71d",
			Description: "initialize node maps in DecodeMsgpack",
			Category:    "IMPORTANT",
			Needed:      "YES",
			Reason:      "Prevents nil map panic during n2n communication",
		},

		// ============= VIEW CHANGE FIXES =============
		{
			Hash:        "cc6ae46d1",
			Description: "prevent orphan MB caching in prevMagicBlock",
			Category:    "VIEW_CHANGE",
			Needed:      "YES",
			Reason:      "Prevents invalid MBs from polluting storage",
		},
		{
			Hash:        "13d263f89",
			Description: "allow Wait() to accept any newer MB",
			Category:    "VIEW_CHANGE",
			Needed:      "YES",
			Reason:      "Fixes view change when MB numbers are not sequential",
		},
		{
			Hash:        "a5891a313",
			Description: "use exact MB lookup in VerifyRelatedMagicBlockPresence",
			Category:    "VIEW_CHANGE",
			Needed:      "YES",
			Reason:      "Fixes verification at view change boundaries",
		},

		// ============= DKG DIAGNOSTICS (SIMPLIFIED) =============
		{
			Hash:        "78f1fd061",
			Description: "simplify DKG diagnostics to backup/restore only",
			Category:    "DKG_DIAG",
			Needed:      "YES",
			Reason:      "Removes complex recovery code, keeps simple backup/restore endpoints",
		},

		// ============= INITIAL CONSENSUS FIXES =============
		{
			Hash:        "037b7d13e",
			Description: "blockchain consensus fixes for stuck chain scenarios",
			Category:    "CONSENSUS",
			Needed:      "REVIEW",
			Reason:      "Large change - need to verify each sub-change is still needed",
		},

		// ============= REVERTED/OBSOLETE =============
		{
			Hash:        "a0cc85a27",
			Description: "Broadcast DKG transactions to all miners + higher fee",
			Category:    "PARTIALLY_REVERTED",
			Needed:      "PARTIAL",
			Reason:      "Higher fee reverted (eee44acf0), broadcast to all miners kept",
		},
		{
			Hash:        "f98fbeead",
			Description: "allow DKG phase recovery when stuck in Unknown state",
			Category:    "REVERTED",
			Needed:      "NO",
			Reason:      "Reverted by eee44acf0 - not needed with Medea hardfork",
		},
		{
			Hash:        "eee44acf0",
			Description: "Revert DKG phase recovery and higher fee priority",
			Category:    "REVERT",
			Needed:      "YES",
			Reason:      "The revert itself is needed",
		},

		// ============= CI/BUILD =============
		{
			Hash:        "074201864",
			Description: "update GitHub Actions versions",
			Category:    "CI",
			Needed:      "YES",
			Reason:      "CI improvements",
		},
		{
			Hash:        "b24078164",
			Description: "remove broken bls submodule reference",
			Category:    "CI",
			Needed:      "YES",
			Reason:      "Build fix",
		},
		{
			Hash:        "072f9ac53",
			Description: "use apt-get instead of apt",
			Category:    "CI",
			Needed:      "YES",
			Reason:      "CI stability",
		},
		{
			Hash:        "3bd55bfb5",
			Description: "stabilize mockery install",
			Category:    "CI",
			Needed:      "YES",
			Reason:      "CI stability",
		},
		{
			Hash:        "624e25751",
			Description: "Revert non-root zchain user from Dockerfiles",
			Category:    "CI",
			Needed:      "YES",
			Reason:      "Docker fix",
		},
		{
			Hash:        "8caedf2e1",
			Description: "Update build-&-publish-docker-image.yml",
			Category:    "CI",
			Needed:      "YES",
			Reason:      "CI update",
		},

		// ============= DKG RECOVERY (REMOVED) =============
		{
			Hash:        "33a97ed4f",
			Description: "add DKG recovery from magic block",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Removed by 78f1fd061 - overcomplicated, not needed",
		},
		{
			Hash:        "fad77799a",
			Description: "use ComputeBlsID helper",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Part of removed DKG recovery feature",
		},
		{
			Hash:        "9f6e2b934",
			Description: "remove DKG verification that breaks loading",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Part of removed DKG recovery feature",
		},
		{
			Hash:        "425f3b327",
			Description: "add ShareOrSigns diagnostics",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Debug code, removed by 78f1fd061",
		},
		{
			Hash:        "81d3c99c3",
			Description: "add endpoint to recover DKG from sharder MB",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Removed by 78f1fd061",
		},
		{
			Hash:        "8ded53385",
			Description: "auto-fetch MB from sharders when DKG recovery fails",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Removed by 78f1fd061",
		},
		{
			Hash:        "888ead20d",
			Description: "correct logging import path",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Fix for removed code",
		},
		{
			Hash:        "0736c1991",
			Description: "correct BackupDKGSummary call signature",
			Category:    "REMOVED",
			Needed:      "NO",
			Reason:      "Fix for removed code",
		},
	}

	// Print summary by category
	categories := map[string][]CommitAnalysis{}
	for _, c := range commits {
		categories[c.Category] = append(categories[c.Category], c)
	}

	order := []string{"ESSENTIAL", "IMPORTANT", "VIEW_CHANGE", "DKG_DIAG", "CONSENSUS",
		"CI", "PARTIALLY_REVERTED", "REVERT", "REMOVED"}

	for _, cat := range order {
		if commits, ok := categories[cat]; ok {
			fmt.Printf("\n### %s ###\n", cat)
			fmt.Println(strings.Repeat("-", 60))
			for _, c := range commits {
				status := "✓"
				if c.Needed == "NO" || c.Needed == "SUPERSEDED" {
					status = "✗"
				} else if c.Needed == "REVIEW" || c.Needed == "PARTIAL" {
					status = "?"
				}
				fmt.Printf("%s [%s] %s\n", status, c.Needed, c.Description)
				fmt.Printf("  Hash: %s\n", c.Hash)
				fmt.Printf("  Reason: %s\n\n", c.Reason)
			}
		}
	}

	// Summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	essential := 0
	keep := 0
	remove := 0
	review := 0

	for _, c := range commits {
		switch c.Needed {
		case "YES":
			keep++
			if c.Category == "ESSENTIAL" || c.Category == "IMPORTANT" {
				essential++
			}
		case "NO", "SUPERSEDED":
			remove++
		case "REVIEW", "PARTIAL":
			review++
		}
	}

	fmt.Printf("\nEssential/Important: %d commits\n", essential)
	fmt.Printf("Other needed:        %d commits\n", keep-essential)
	fmt.Printf("To review:           %d commits\n", review)
	fmt.Printf("Can remove:          %d commits (already removed or superseded)\n", remove)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("COMMITS NEEDED FOR RECOVERY (in order of importance)")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println(`
ESSENTIAL (Must have):
  1. UNCOMMITTED: ViewChangeOffset = 20
  2. UNCOMMITTED: reachedNotarization always checks threshold

IMPORTANT (Stability fixes):
  3. c229e44ce: move SetRandomSeed after verification
  4. 7a97add24: allow restart of rounds in Complete phase
  5. a3a2f3631: load previous MB's DKG on startup
  6. 28a53c71d: initialize node maps in DecodeMsgpack

VIEW CHANGE:
  7. cc6ae46d1: prevent orphan MB caching
  8. 13d263f89: allow Wait() to accept any newer MB
  9. a5891a313: use exact MB lookup in VerifyRelatedMagicBlockPresence

OTHER:
  10. 78f1fd061: simplify DKG diagnostics (backup/restore only)
  11. eee44acf0: Revert DKG phase recovery (needed)
  12. All CI commits (keep for build stability)

ALREADY HANDLED (code removed by later commits):
  - 33a97ed4f, fad77799a, 9f6e2b934, 425f3b327, 81d3c99c3, 8ded53385, 888ead20d, 0736c1991
  - These commits added DKG recovery features that were later removed by 78f1fd061

NEEDS REVIEW:
  - 037b7d13e: Large initial consensus fix - verify each change is still needed
`)
}
