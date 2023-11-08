package util

import (
	"math/rand"
	"time"
)

// ShuffleString shuffles a string of characters and returns the result
func ShuffleString(str string) string {
	src := rand.NewSource(time.Now().Unix())
	r := rand.New(src)

	inRune := []rune(str)
	r.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}
