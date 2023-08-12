package utils

import (
	"0chain.net/core/encryption"
)

var signature = encryption.NewBLS0ChainScheme()

func init() {
	if err := signature.GenerateKeys(); err != nil {
		panic(err)
	}
}

// Sign by internal ("wrong") secret key generated randomly once client created.
func Sign(hash string) (sign string, err error) {
	return signature.Sign(hash)
}

func SliceDifference[T comparable](s1, s2 []T) []T {
	mp := make(map[T]interface{})
	for _, v := range s2 {
		mp[v] = nil
	}

	out := make([]T, 0, len(s1))
	for _, v := range s1 {
		if _, ok := mp[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}

func SliceUnion[T comparable](s1, s2 []T) []T {
	newCap := len(s1) + len(s2)
	found := make(map[T]interface{})
	out := make([]T, 0, newCap)

	for _, v := range s1 {
		if _, ok := found[v]; ok {
			continue
		}
		found[v] = nil
		out = append(out, v)
	}

	for _, v := range s2 {
		if _, ok := found[v]; ok {
			continue
		}
		found[v] = nil
		out = append(out, v)
	}

	return out
}