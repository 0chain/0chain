package round

import "0chain.net/block"

/*Round - data structure for the round */
type Round struct {
	Number int64
	Role   int
	Block  *block.Block
}

/*RoleGenerator - block genreator role */
var RoleGenerator = 1

/*RoleVerifier - block verifier role */
var RoleVerifier = 2
