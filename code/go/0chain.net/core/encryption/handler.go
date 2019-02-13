package encryption

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
)

/*HashHandler - returns hash of the text passed */
func HashHandler(w http.ResponseWriter, r *http.Request) {
	text := r.FormValue("text")
	if text == "" {
		var data []byte
		buff, err := r.Body.Read(data)
		if err != nil {
			return
		}
		text = string(buff)
	}
	fmt.Fprintf(w, Hash(text))
}

/*SignHandler - returns hash of the text passed */
func SignHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	privateKey := r.FormValue("private_key")
	publicKey := r.FormValue("public_key")
	data := r.FormValue("data")
	timestamp := r.FormValue("timestamp")
	key, err := hex.DecodeString(publicKey)
	if err != nil {
		return nil, err
	}
	clientID := Hash(key)
	var hashdata string
	if timestamp != "" {
		hashdata = fmt.Sprintf("%v:%v:%v", clientID, timestamp, data)
	} else {
		hashdata = fmt.Sprintf("%v:%v", clientID, data)
	}
	hash := Hash(hashdata)
	signature, err := Sign(privateKey, hash)
	if err != nil {
		return nil, err
	}
	json := make(map[string]interface{})
	json["client_id"] = clientID
	json["hash"] = hash
	json["signature"] = signature
	return json, nil
}
