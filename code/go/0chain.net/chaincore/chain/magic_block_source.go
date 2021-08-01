package chain

import (
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"

	zchainErrors "github.com/0chain/gosdk/errors"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/httpclientutil"

	"0chain.net/core/logging"
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
		return "", zchainErrors.Newf("", "invalid 0DNS URL base: %v", err)
	}
	// join to given base can end with '/v1/', for example
	full.Path = path.Join(full.Path, zdnsMagicBlockEndpoint)
	return full.String(), nil
}

// ReadMagicBlockFile obtains MB from JSON file with given path.
func ReadMagicBlockFile(path string) (mb *block.MagicBlock, err error) {

	if path == "" {
		return nil, zchainErrors.New("empty magic block file path")
	}

	if ext := filepath.Ext(path); ext != ".json" {
		return nil, zchainErrors.Newf("", "unexpected magic block file extension: %q, "+
			"expected '.json'", ext)
	}

	var b []byte
	if b, err = ioutil.ReadFile(path); err != nil {
		return nil, zchainErrors.Newf("", "reading magic block file: %v", err)
	}

	mb = block.NewMagicBlock()
	if err = mb.Decode(b); err != nil {
		return nil, zchainErrors.Newf("", "decoding magic block file: %v", err)
	}

	logging.Logger.Info("read magic block file",
		zap.Any("number", mb.MagicBlockNumber),
		zap.Any("sr", mb.StartingRound),
		zap.Any("hash", mb.Hash))
	return
}

// GetMagicBlockFrom0DNS with given URL base.
func GetMagicBlockFrom0DNS(urlBase string) (mb *block.MagicBlock, err error) {
	if urlBase == "" {
		return nil, zchainErrors.New("empty 0DNS URL base configured")
	}
	var full string
	if full, err = get0DNSMagicBlockEndpoint(urlBase); err != nil {
		return nil, zchainErrors.Newf("", "0DNS URL error: %v", err)
	}
	mb = block.NewMagicBlock()
	if err = httpclientutil.MakeGetRequest(full, mb); err != nil {
		return nil, zchainErrors.Newf("", "getting MB from 0DNS %q: %v", full, err)
	}
	logging.Logger.Info("get magic block file from 0DNS", zap.String("0dns", full),
		zap.Any("number", mb.MagicBlockNumber),
		zap.Any("sr", mb.StartingRound),
		zap.Any("hash", mb.Hash))
	return
}
