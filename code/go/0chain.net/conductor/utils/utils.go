package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

func CopyDir(src string, dst string) error {
    entries, err := os.ReadDir(src)
    if err != nil {
        return err
    }

    err = os.MkdirAll(dst, 0755)
    if err != nil {
        return err
    }

    for _, entry := range entries {
        srcPath := filepath.Join(src, entry.Name())
        dstPath := filepath.Join(dst, entry.Name())

        if entry.IsDir() {
            err = CopyDir(srcPath, dstPath)
            if err != nil {
                return err
            }
        } else {
            err = CopyFile(srcPath, dstPath)
            if err != nil {
                return err
            }
        }
    }

    return nil
}

func CopyFile(src string, dst string) error {
    srcFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer srcFile.Close()

    dstFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer dstFile.Close()

    _, err = io.Copy(dstFile, srcFile)
    if err != nil {
        return err
    }

    return nil
}

func FileNamify(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ":", "(colon)")
	s = strings.ReplaceAll(s, "'", "(single-quote)")
	s = strings.ReplaceAll(s, "\"", "(double-quote)")
	s = strings.ReplaceAll(s, "/", "(forward-slash)")
	s = strings.ReplaceAll(s, "\\", "(back-slash)")
	s = strings.ReplaceAll(s, "?", "(question-mark)")
	s = strings.ReplaceAll(s, "*", "(asterisk)")
	s = strings.ReplaceAll(s, "<", "(left-angle-bracket)")
	s = strings.ReplaceAll(s, ">", "(right-angle-bracket)")
	s = strings.ReplaceAll(s, "|", "(pipe)")
	s = strings.ReplaceAll(s, "&", "(ampersand)")
	s = strings.ReplaceAll(s, "%", "(percent)")
	s = strings.ReplaceAll(s, "$", "(dollar-sign)")
	s = strings.ReplaceAll(s, "#", "(hash)")
	s = strings.ReplaceAll(s, "@", "(at-sign)")
	s = strings.ReplaceAll(s, "!", "(exclamation-mark)")
	s = strings.ReplaceAll(s, "`", "(backtick)")
	s = strings.ReplaceAll(s, "+", "(plus-sign)")
	s = strings.ReplaceAll(s, "=", "(equals-sign)")
	s = strings.ReplaceAll(s, "{", "(left-curly-prace)")
	s = strings.ReplaceAll(s, "}", "(right-curly-prace)")
	s = strings.ReplaceAll(s, "\n", "-")
	s = strings.ReplaceAll(s, "\r", "-")
	s = strings.ReplaceAll(s, "\t", "-")
	s = strings.ReplaceAll(s, "\v", "-")
	s = strings.ReplaceAll(s, "\f", "-")
	s = strings.ReplaceAll(s, "\b", "-")
	s = strings.ReplaceAll(s, "\a", "-")
	return s
}
