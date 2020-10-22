package chain

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/httpclientutil"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

// The get0DNSMagicBlockEndpoint returns full URL to get MB from a 0DNS server
// giving only server address (like http://127.0.0.1:9091 ->
// http://127.0.0.1:9091/dns/magic_lock)
func get0DNSMagicBlockEndpoint(base string) (ep string, err error) {
	// the zdnsMagicBlockEndpoint is 0DNS endpoint
	// to get latest known magic block
	const zdnsMagicBlockEndpoint = "/magic_block"

	var full *url.URL
	if full, err = url.Parse(base); err != nil {
		return "", fmt.Errorf("invalid 0DNS URL base: %v", err)
	}
	// join to given base can end with '/v1/', for example
	full.Path = path.Join(full.Path, zdnsMagicBlockEndpoint)
	return full.String(), nil
}

// ReadMagicBlockFile obtains MB from JSON file with given path.
func ReadMagicBlockFile(path string) (mb *block.MagicBlock, err error) {

	if path == "" {
		return nil, errors.New("empty magic block file path")
	}

	if ext := filepath.Ext(path); ext != ".json" {
		return nil, fmt.Errorf("unexpected magic block file extension: %q, "+
			"expected '.json'", ext)
	}

	var b []byte
	if b, err = ioutil.ReadFile(path); err != nil {
		return nil, fmt.Errorf("reading magic block file: %v", err)
	}

	mb = block.NewMagicBlock()
	if err = mb.Decode(b); err != nil {
		return nil, fmt.Errorf("decoding magic block file: %v", err)
	}

	Logger.Info("read magic block file", zap.Any("mb", mb))
	return
}

// GetMagicBlockFrom0DNS with given URL base.
func GetMagicBlockFrom0DNS(urlBase string) (mb *block.MagicBlock, err error) {
	if urlBase == "" {
		return nil, errors.New("empty 0DNS URL base configured")
	}
	var full string
	if full, err = get0DNSMagicBlockEndpoint(urlBase); err != nil {
		return nil, fmt.Errorf("0DNS URL error: %v", err)
	}
	mb = block.NewMagicBlock()
	if err = httpclientutil.MakeGetRequest(full, mb); err != nil {
		return nil, fmt.Errorf("getting MB from 0DNS %q: %v", full, err)
	}
	Logger.Info("get magic block file from 0DNS", zap.Any("mb", mb),
		zap.String("0dns", full))
	return
}
