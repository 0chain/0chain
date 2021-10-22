package handlers

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"strconv"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/core/logging"
	"0chain.net/core/viper"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/0chain/errors"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

const updateConfigURL = "/v1/config/update"
const updateConfigAllURL = "/v1/config/update_all"

// HTML Form
const formHTML = `
<html>
<head>
    <title>Update Config</title>
</head>
<body>
    <form action='{{.URL}}' method='post'>
        Generation Timeout (time till a miner makes a block with less than max blocksize): <input type='text' name='generate_timeout' value='{{.GenerateTimeout}}'><br>
        Retry Wait Time (time miner waits if there aren't enough transactions to reach max blocksize): <input type='text' name='txn_wait_time' value='{{.TnxWaitTime}}'><br>
        <input type='submit' value='Submit'>
    </form>
</body>
</html>
`

// ConfigUpdate - "/v1/config/update" ConfigUpdate
func (m *minerGRPCService) ConfigUpdate(_ context.Context, req *minerproto.ConfigUpdateRequest) (*httpbody.HttpBody, error) {
	output, err := updateConfig(req.GenerateTimeout, req.TxnWaitTime, updateConfigURL)
	if err != nil {
		return nil, err
	}

	return &httpbody.HttpBody{
		ContentType: "text/html;charset=UTF-8",
		Data:        output,
	}, nil
}

// ConfigUpdateAll - "/v1/config/update_all"
func (m *minerGRPCService) ConfigUpdateAll(_ context.Context, req *minerproto.ConfigUpdateRequest) (*httpbody.HttpBody, error) {
	mb := chain.GetServerChain().GetCurrentMagicBlock()

	// range all miners
	for _, miner := range mb.Miners.Nodes {
		if node.Self.Underlying().PublicKey != miner.PublicKey {
			go func(miner *node.Node) {
				form := map[string][]string{
					"generate_timeout": []string{req.GenerateTimeout},
					"txn_wait_time":    []string{req.TxnWaitTime},
				}
				resp, err := http.PostForm(miner.GetN2NURLBase()+updateConfigURL, form)
				if err != nil {
					logging.Logger.Error("failed to update other miner's config", zap.Any("miner", miner.GetKey()), zap.Any("response", resp), zap.Any("error", err))
					return
				}
				defer resp.Body.Close()
			}(miner)
		}
	}
	updateConfig(req.GenerateTimeout, req.TxnWaitTime, updateConfigAllURL)
	//
	return nil, nil
}

// updateConfig
func updateConfig(genTimeout, tnxWaitTime, updateURL string) ([]byte, error) {
	newGenTimeout, _ := strconv.Atoi(genTimeout)
	if newGenTimeout > 0 {
		chain.GetServerChain().SetGenerationTimeout(newGenTimeout)
		viper.Set("server_chain.block.generation.timeout", newGenTimeout)
	}
	newTxnWaitTime, _ := strconv.Atoi(tnxWaitTime)
	if newTxnWaitTime > 0 {
		chain.GetServerChain().SetRetryWaitTime(newTxnWaitTime)
		viper.Set("server_chain.block.generation.retry_wait_time", newTxnWaitTime)
	}

	// parse HTML form
	tmpl, err := template.New("html_form").Parse(formHTML)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse html form")
	}

	// data to insert in the HTML form
	var params = struct {
		URL, GenerateTimeout, TnxWaitTime string
	}{
		URL:             updateURL,
		GenerateTimeout: viper.Get("server_chain.block.generation.timeout").(string),
		TnxWaitTime:     viper.Get("server_chain.block.generation.retry_wait_time").(string),
	}

	// execute tmpl and generate HTML form.
	var output bytes.Buffer
	if err := tmpl.Execute(&output, params); err != nil {
		return nil, errors.Wrap(err, "could not execute html form")
	}

	// return html parsed form
	return output.Bytes(), nil
}
