### Migrate `/v1/config/update` to grpc #558

*note: `ConfigUpdateAllHandler` same as update but for all miner nodes*


- current handler uses http.ResponseWriter to directly write HTML in the browser. see:
  `code/go/0chain.net/miner/miner/handler.go`
  `func updateConfig(w http.ResponseWriter, r *http.Request, updateUrl string)`

- on the http handler it extract:
  `r.FormValue("generate_timeout")`
  `r.FormValue("txn_wait_time")`
  and generate new config, also updates new config in viper

- `POST` method see:
   `fmt.Fprintf(w, "<form action='%s' method='post'>", updateUrl)`
   `resp, err := http.PostForm(miner.GetN2NURLBase()+updateConfigURL, r.Form)`


```proto
// request
message UpdateConfigRequest {
    string generate_timeout = 1;
    string txn_wait_time = 2;
}

// response
message UpdateConfigResponse {
    // todo
}
```

- some requests uses HTTP header to get informations, such as: `nodeID := r.Header.Get(HeaderNodeID)`
- different content type will be rejected: `contentType := r.Header.Get("Content-type")` - see: node.ToN2NReceiveEntityHandler
- reqSignature := r.Header.Get(HeaderNodeRequestSignature)
- entityName := r.Header.Get(HeaderRequestEntityName)
- entityID := r.Header.Get(HeaderRequestEntityID)
- chainID := r.Header.Get(HeaderRequestChainID)
- reqTS := r.Header.Get(HeaderRequestTimeStamp)
- reqHash := r.Header.Get(HeaderRequestHash)