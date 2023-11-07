package utils

import (
	"fmt"
	"io"
	"net/http"

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

// StringSlice casts all elements of the array to string. All elements need to be in simple type (numeric, string)
func StringSlice(s []interface{}) ([]string, error) {
	result := make([]string, 0, len(s))
	for i, el := range s {
		switch tel := el.(type) {
		case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
			result = append(result, fmt.Sprintf("%v", tel))
		default:
			return nil, fmt.Errorf("error in element %v (%v). Not a convertable type", i, el)
		}
	}

	return result, nil
}

func HttpGet(url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Error in GET request to url %v", url)
	}

	bdy, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bdy, nil
}
