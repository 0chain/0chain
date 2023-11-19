package encryption

// HashScorer - takes two hashes and provides a score
type HashScorer interface {
	Score(hash1 []byte, hash2 []byte) int32
}

// XORHashScorer - a scorer based on XOR operation
type XORHashScorer struct {
}

// NewXORHashScorer - create a new XOR based hash scorer
func NewXORHashScorer() *XORHashScorer {
	return &XORHashScorer{}
}

// Score - implement interface
func (xor *XORHashScorer) Score(hash1 []byte, hash2 []byte) int32 {
	var score int32
	for idx, b := range hash1 {
		x := b ^ hash2[idx]
		for i := 0; i < 8; i++ {
			score += int32(x & 1)
			x >>= 1
		}
	}
	return score
}
