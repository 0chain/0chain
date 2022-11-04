package util

import (
	"math/rand"
	"time"
)

// ShuffleString shuffles a string of characters and returns the result
func ShuffleString(str string) string {
	rand.Seed(time.Now().Unix())
	inRune := []rune(str)
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}
