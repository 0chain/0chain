package encryption

import (
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
