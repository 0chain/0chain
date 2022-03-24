package chain

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"github.com/stretchr/testify/require"
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

			c.On("GetLatestFinalizedMagicBlockClone", req.Context()).Return(tc.retLFMB)
			handler := common.ToJSONResponse(LatestFinalizedMagicBlockHandler(&c))

			w := httptest.NewRecorder()
			handler(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tc.expectCode, resp.StatusCode)

			if tc.expectCode == http.StatusNotModified {
				d, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Empty(t, d)
				return
			}

			// decode the body and compare the
			b := &block.Block{}
			d, err := ioutil.ReadAll(resp.Body)
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
	//d, err := ioutil.ReadAll(resp.Body)
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

		body, err := ioutil.ReadAll(w.Result().Body)

		wantSubstring := `I am Miner000 working on the chain`

		require.NoError(t, err)
		require.Contains(t, string(body), wantSubstring)
		require.Equal(t, 200, w.Result().StatusCode)
	})

	t.Run("request to non-root path must return 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://localhost:7071/unknown", nil)
		w := httptest.NewRecorder()

		HomePageAndNotFoundHandler(w, req)

		body, err := ioutil.ReadAll(w.Result().Body)

		wantSubstring := `{"code":"resource_not_found","error":"resource_not_found: can't retrieve resource"}`

		require.NoError(t, err)
		require.Contains(t, string(body), wantSubstring)
		require.Equal(t, 404, w.Result().StatusCode)
	})
}
