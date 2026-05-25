// gapfinder compares block hashes in postgres against a sorted hash list
// to find blocks missing from packs.
//
// Usage:
//   gapfinder -db "host=localhost user=zchain_user dbname=events_db sslmode=disable" \
//             -hashes /path/to/hashes.txt \
//             -maxround 153000000 \
//             > missing.txt
//
// Output: round<TAB>hash per line for each missing block.
// Also prints summary stats to stderr.
package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dbConn := flag.String("db", "", "postgres connection string (required)")
	hashFile := flag.String("hashes", "", "sorted hash list from hashexport (required)")
	maxRound := flag.Int64("maxround", 0, "only check blocks up to this round (0 = all)")
	flag.Parse()

	if *dbConn == "" || *hashFile == "" {
		fmt.Fprintln(os.Stderr, "error: -db and -hashes flags required")
		flag.Usage()
		os.Exit(1)
	}

	// Load hash set from file
	start := time.Now()
	fmt.Fprintln(os.Stderr, "Loading hash set...")
	hashSet, err := loadHashSet(*hashFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading hashes: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d hashes in %v\n", len(hashSet), time.Since(start).Round(time.Second))

	// Connect to postgres
	db, err := sql.Open("postgres", *dbConn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Query blocks
	query := "SELECT round, hash FROM blocks ORDER BY round ASC"
	if *maxRound > 0 {
		query = fmt.Sprintf("SELECT round, hash FROM blocks WHERE round <= %d ORDER BY round ASC", *maxRound)
	}

	fmt.Fprintln(os.Stderr, "Querying blocks from postgres...")
	rows, err := db.Query(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error querying: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	w := bufio.NewWriterSize(os.Stdout, 256*1024)
	total := 0
	missing := 0

	for rows.Next() {
		var round int64
		var hash string
		if err := rows.Scan(&round, &hash); err != nil {
			fmt.Fprintf(os.Stderr, "error scanning: %v\n", err)
			os.Exit(1)
		}
		total++
		if _, ok := hashSet[hash]; !ok {
			fmt.Fprintf(w, "%d\t%s\n", round, hash)
			missing++
		}
		if total%5000000 == 0 {
			fmt.Fprintf(os.Stderr, "  checked %dM blocks, %d missing so far...\n", total/1000000, missing)
		}
	}
	w.Flush()

	elapsed := time.Since(start).Round(time.Second)
	fmt.Fprintf(os.Stderr, "Done. %d total blocks, %d in packs, %d missing (%.1f%%) in %v\n",
		total, total-missing, missing, float64(missing)*100/float64(total), elapsed)
}

func loadHashSet(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	set := make(map[string]struct{}, 30000000) // pre-size for ~30M
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		set[scanner.Text()] = struct{}{}
	}
	return set, scanner.Err()
}
