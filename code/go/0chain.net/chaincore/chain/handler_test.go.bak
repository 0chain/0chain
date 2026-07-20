package chain

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io"

	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGetLatestFinalizedMagicBlock(t *testing.T) {
	lfmb := &block.Block{}
	lfmb.Hash = "abcd"

	lfmb2 := &block.Block{}
	lfmb2.Hash = "cdef"

	tt := []struct {
		name       string
		localLFMB  string
		retLFMB    *block.Block
		expectCode int
	}{
		{
			name:       "not modified, set node lfmb",
			localLFMB:  lfmb.Hash,
			retLFMB:    lfmb,
			expectCode: http.StatusNotModified,
		},
		{
			name:       "not modified, no node lfmb",
			retLFMB:    lfmb,
			expectCode: http.StatusOK,
		},
		{
			name:       "modified, no node lfmb",
			retLFMB:    lfmb2,
			expectCode: http.StatusOK,
		},
		{
			name:       "modified, set node lfmb",
			localLFMB:  lfmb.Hash,
			retLFMB:    lfmb2,
			expectCode: http.StatusOK,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := MockChainer{}

			var data io.Reader
			if len(tc.localLFMB) > 0 {
				params := url.Values{}
				params.Add("node-lfmb-hash", tc.localLFMB)
				data = strings.NewReader(params.Encode())
			}

			req := httptest.NewRequest("POST", "/v1/block/get/latest_finalized_magic_block", data)
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

			c.On("GetCurrentRound").Return(int64(1))
			c.On("GetMagicBlock", int64(1)).Return(&block.MagicBlock{})

			c.On("GetLatestFinalizedMagicBlockClone", req.Context()).Return(tc.retLFMB)
			handler := common.ToJSONResponse(LatestFinalizedMagicBlockHandler(&c))

			w := httptest.NewRecorder()
			handler(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tc.expectCode, resp.StatusCode)

			if tc.expectCode == http.StatusNotModified {
				d, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Empty(t, d)
				return
			}

			// decode the body and compare the
			b := &block.Block{}
			d, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, b.Decode(d))
			require.Equal(t, tc.retLFMB.Hash, b.Hash)
		})
	}

	//c := MockChainer{}
	//
	//lfmb := block.Block{}
	//lfmb.Hash = "abcd"
	//req := httptest.NewRequest("GET", "/v1/block/get/latest_finalized_magic_block", nil)
	//req.Header.Set(node.HeaderNodeLFMBHash, "abcd")
	//
	//c.On("GetLatestFinalizedMagicBlockClone", req.Context()).Return(&lfmb)
	//handler := common.ToJSONResponse(LatestFinalizedMagicBlockHandler(&c))
	//
	//w := httptest.NewRecorder()
	//handler(w, req)
	//resp := w.Result()
	//require.Equal(t, http.StatusNotModified, resp.StatusCode)
	//
	//// modified
	//req = httptest.NewRequest("GET", "/v1/block/get/latest_finalized_magic_block", nil)
	//handler(w, req)
	//resp = w.Result()
	//defer resp.Body.Close()
	//d, err := io.ReadAll(resp.Body)
	//require.NoError(t, err)
	//
	//b := block.Block{}
	//
	//fmt.Println(string(d))
	//err = b.Decode(d)
	//require.NoError(t, err)
	//require.Equal(t, lfmb.Hash, b.Hash)

}

func TestHomePageAndNotFoundHandler(t *testing.T) {
	t.Run("request to root path must return home page", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://localhost:7071/", nil)
		w := httptest.NewRecorder()

		c := Provider().(*Chain)
		SetServerChain(c)
		defer SetServerChain(nil) // ensure to reset after test

		HomePageAndNotFoundHandler(w, req)

		body, err := io.ReadAll(w.Result().Body)

		wantSubstring := `I am Miner000 working on the chain`

		require.NoError(t, err)
		require.Contains(t, string(body), wantSubstring)
		require.Equal(t, 200, w.Result().StatusCode)
	})

	t.Run("request to non-root path must return 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://localhost:7071/unknown", nil)
		w := httptest.NewRecorder()

		HomePageAndNotFoundHandler(w, req)

		body, err := io.ReadAll(w.Result().Body)

		wantSubstring := `{"code":"resource_not_found","error":"resource_not_found: can't retrieve resource"}`

		require.NoError(t, err)
		require.Contains(t, string(body), wantSubstring)
		require.Equal(t, 404, w.Result().StatusCode)
	})
}

func makeTestNode() (*node.Node, error) {
	ss := encryption.NewBLS0ChainScheme()
	ss.GenerateKeys()
	pbK := ss.GetPublicKey()

	nc := map[interface{}]interface{}{
		"type":       node.NodeTypeMiner,
		"public_ip":  "public ip",
		"n2n_ip":     "n2n_ip",
		"port":       8080,
		"id":         util.ToHex([]byte(pbK)),
		"public_key": pbK,
	}
	return node.NewNode(nc)
}

func generateProposedBlockToRound(t *testing.T, r *round.Round, n *node.Node) {
	b := block.NewBlock("", r.Number)
	b.MinerID = n.Client.ID
	randomData := make([]byte, 10)
	read, err := rand.Reader.Read(randomData)
	require.NoError(t, err)
	require.Equal(t, read, len(randomData))
	txn := transaction.Transaction{HashIDField: datastore.HashIDField{Hash: encryption.Hash(randomData)}}
	b.Txns = append(b.Txns, &txn)
	b.AddTransaction(&txn)
	b.TxnsMap = make(map[string]bool)
	b.TxnsMap[txn.Hash] = true
	b.HashBlock()
	r.AddProposedBlock(b)
}

func TestRoundInfoHandler(t *testing.T) {
	runRequest := func(c Chainer) (body string) {
		var err error
		req, _ := http.NewRequest(http.MethodGet, "/_diagnostics/round_info", nil)
		w := httptest.NewRecorder()
		var loggedMessages bytes.Buffer
		writer := bufio.NewWriter(&loggedMessages)
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(writer),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return true
			}),
		)
		logging.Logger = zap.New(core, zap.Development())
		RoundInfoHandler(c)(w, req)
		bodybytes, err := io.ReadAll(w.Result().Body)
		require.NoError(t, err)
		require.Equal(t, 200, w.Result().StatusCode)
		err = writer.Flush()
		require.NoError(t, err)
		body = string(bodybytes)
		require.NotContains(t, loggedMessages.String(), `DPANIC`)
		return
	}

	blocksSubstring := `Block Verification and Notarization`
	vrfSubstring := `VRF Shares`

	c := MockChainer{}
	c.On("GetCurrentRound").Return(int64(1))
	mb := block.NewMagicBlock()
	mb.MagicBlockNumber = 1
	mb.Miners = node.NewPool(1)
	n1, err := makeTestNode()
	require.NoError(t, err)
	mb.Miners.AddNode(n1)
	n2, err := makeTestNode()
	require.NoError(t, err)
	mb.Miners.AddNode(n2)
	r1 := &round.Round{Number: 1}
	generateProposedBlockToRound(t, r1, n1)
	generateProposedBlockToRound(t, r1, n2)

	c.On("GetRound", int64(1)).Return(r1)
	c.On("GetMagicBlock", int64(1)).Return(mb)

	// call RoundInfoHandler on a round without seed and ranks
	body := runRequest(&c)
	require.Contains(t, body, blocksSubstring)
	require.NotContains(t, body, vrfSubstring)

	r1.SetRandomSeed(time.Now().UnixNano(), 2)

	// call RoundInfoHandler on a round with seed and ranks
	body = runRequest(&c)
	require.Contains(t, string(body), blocksSubstring)
	require.Contains(t, string(body), vrfSubstring)

}
