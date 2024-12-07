package transaction

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"0chain.net/chaincore/node"
	"0chain.net/miner"
	"github.com/pkg/errors"
)

// PerformZauthSignTxn performs zauth transaction sign call.
func PerformZauthSignTxn(signature string) (string, error) {
	mc := miner.GetMinerChain()

	req, err := http.NewRequest(
		"POST", mc.ChainConfig.ZauthServer()+"/sign/txn", bytes.NewBuffer([]byte(signature)))
	if err != nil {
		return "", errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Peer-Public-Key", node.Self.PublicKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to send HTTP request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rsp, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrap(err, "failed to read response body")
		}

		return "", errors.Errorf("unexpected status code: %d, res: %s", resp.StatusCode, string(rsp))
	}

	var d string

	err = json.NewDecoder(resp.Body).Decode(&d)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	return d, nil
}
