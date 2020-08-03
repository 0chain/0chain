package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
)

const (
	UPLOAD   = "/v1/file/upload/"
	COMMIT   = "/v1/connection/commit/"
	DOWNLOAD = "/v1/file/download/"
)

var markersOrders *appOrders

type waitOrder struct {
	sent chan struct{}            // request sent -> next order
	wait map[string]chan struct{} // marker -> release
}

func newWaitOrder() (wo *waitOrder) {
	wo = new(waitOrder)
	wo.sent = make(chan struct{})
	wo.wait = make(map[string]chan struct{})
	return
}

// run in goroutine
func (wo *waitOrder) perform(orders []string) {
	println("PERFORM", fmt.Sprint(orders))
	for _, marker := range orders {
		var release = wo.wait[marker]
		close(release)
		println("RELEASE", marker)
		<-wo.sent // wait maker sent report and release next
	}
}

// application orders controller
type appOrders struct {
	sync.Mutex

	orders   []string              // set by flags
	blobbers map[string]*waitOrder // collected by incoming requests

	// service channels
	draining chan struct{}
	unlocked chan struct{}
}

func newAppOrder(orders []string) (ao *appOrders) {
	ao = new(appOrders)
	ao.orders = orders
	ao.blobbers = make(map[string]*waitOrder)
	ao.draining = make(chan struct{})
	ao.unlocked = make(chan struct{})
	close(ao.unlocked)
	go ao.drain()
	return
}

// add blobber to the orders
func (ao *appOrders) addBlobber(host string) (wo *waitOrder) {
	var ok bool
	if wo, ok = ao.blobbers[host]; ok {
		return // already have
	}
	println("add blobber", host)
	wo = newWaitOrder()
	ao.blobbers[host] = wo
	return // nil
}

func (ao *appOrders) drain() {
	for range ao.draining {
	}
}

func (ao *appOrders) addOrder(host, marker string) (sent, rel chan struct{}) {
	println("add order", host, marker)

	if len(ao.orders) == 0 {
		println("add order", host, marker, "NO ORDERS")
		return ao.draining, ao.unlocked // released
	}

	ao.Lock()
	defer ao.Unlock()

	var (
		wo = ao.addBlobber(host)
		ok bool
	)

	if rel, ok = wo.wait[marker]; ok {
		println("add order", host, marker, "ALREADY HAVE")
		return wo.sent, rel
	}

	println("add order", host, marker, "ADD")

	rel = make(chan struct{})
	wo.wait[marker] = rel

	if len(wo.wait) == len(ao.orders) {
		println("add order", host, marker, "PERMORM")
		go wo.perform(ao.orders) // have all required request in the queue
		// delete(ao.blobbers, host) // and remove the reference
	}

	return wo.sent, rel
}

// split by "-"
func parseOrder(order string) (ords []string, err error) {
	ords = strings.Split(order, "-")
	for _, s := range ords {
		switch s {
		case "dm", "wm", "rm":
		default:
			return nil, fmt.Errorf("unknown marker name: %q", s)
		}
	}
	return
}

func isDeleteMarker(r *http.Request) (ok bool) {
	var val = r.FormValue("write_marker")

	type writeMarker struct {
		Size int64 `json:"size"`
	}

	var wm writeMarker
	if err := json.Unmarshal([]byte(val), &wm); err != nil {
		log.Print("[ERR] parsing write_maker: ", err)
	}

	return wm.Size < 0
}

// regardless a filter
func hasField(r *http.Request, field string) (ok bool) {
	for name := range r.MultipartForm.File {
		if name == field {
			return true
		}
	}
	for name := range r.MultipartForm.Value {
		if name == field {
			return true
		}
	}
	return false
}

func skipFormField(skip string, r *http.Request) (q *http.Request, err error) {

	r.ParseMultipartForm(0)
	if r.MultipartForm == nil {
		return r, nil
	}

	// copy multipart form
	var (
		body bytes.Buffer
		mp   = multipart.NewWriter(&body)
	)

	// copy files
	for name, files := range r.MultipartForm.File {
		if name == skip {
			log.Print("skip field:", name, "(file)")
			continue
		}

		for _, fh := range files {
			println("-", name, ":", fh.Filename, fh.Size)
			var file io.Writer
			if file, err = mp.CreateFormFile(name, fh.Filename); err != nil {
				log.Print("creating mp file: ", err)
				return
			}
			var got multipart.File
			if got, _, err = r.FormFile(name); err != nil {
				log.Print("getting mp file: ", err)
				return
			}
			if _, err = io.Copy(file, got); err != nil {
				log.Print("copying mp file: ", err)
				return
			}
		}
	}

	// copy fields
	for name, values := range r.MultipartForm.Value {
		if name == skip {
			log.Print("skip field:", name)
			continue
		}

		for _, val := range values {
			println("-", name, ":", val)
			if err = mp.WriteField(name, val); err != nil {
				log.Print("writing mp field: ", err)
				return
			}
		}
	}

	if err = mp.Close(); err != nil {
		log.Print("closing mp:", err)
		return
	}

	q, err = http.NewRequest(r.Method, r.URL.String(), &body)
	if err != nil {
		log.Print("creating request: ", err)
		return
	}

	q.Header.Add("Content-Type", mp.FormDataContentType())
	q.Header.Set("X-App-Client-ID", r.Header.Get("X-App-Client-ID"))
	q.Header.Set("X-App-Client-Key", r.Header.Get("X-App-Client-Key"))

	return q, nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func simpleRoundTrip(w http.ResponseWriter, r *http.Request) {
	var resp, err = http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handle(w http.ResponseWriter, r *http.Request, markers, filter string) {

	if !strings.HasPrefix(r.URL.Host, "localhost") {
		simpleRoundTrip(w, r) // not for a blobber
		return
	}

	var (
		q   *http.Request
		err error
	)

	switch {

	case strings.Contains(r.URL.Path, UPLOAD):
		log.Print("[INF] upload: ", r.URL.String())
		q, err = skipFormField(filter, r)

	case strings.Contains(r.URL.Path, COMMIT),
		strings.Contains(r.URL.Path, DOWNLOAD):

		log.Print("[INF] commit: ", r.URL.String())
		q, err = skipFormField(filter, r) // copy request and parse multipart

		if err != nil {
			break
		}

		var sent, release chan struct{}
		switch {
		case hasField(r, "write_marker"):
			if isDeleteMarker(r) {
				println("GOT DELETE MARKER")
				sent, release = markersOrders.addOrder(r.URL.Host, "dm")
			} else {
				println("GOT WRITE MARKER")
				sent, release = markersOrders.addOrder(r.URL.Host, "wm")
			}
		case hasField(r, "read_marker"):
			println("GOT READ MARKER")
			sent, release = markersOrders.addOrder(r.URL.Host, "rm")
		}

		// send on release
		println(":::: WAIT RELEASING")
		defer func(sent chan struct{}) { sent <- struct{}{} }(sent)
		<-release
		println(":::: RELEASED")

	default:
		log.Print("[INF] forward: ", r.URL.String())
		q = r

	}

	if err != nil {
		log.Print("[ERR] ", err)
		http.Error(w, err.Error(), 500)
		return
	}

	simpleRoundTrip(w, q)
}

func main() {

	// address
	var (
		markers string      = ""              // markers arriving order
		filter  string      = ""              // filter multipart forms fields
		addr    string      = "0.0.0.0:15211" // bind
		s       http.Server                   // server instance
	)

	flag.StringVar(&markers, "m", markers, "markers arriving order")
	flag.StringVar(&filter, "f", filter, "filter multipart form fields")
	flag.StringVar(&addr, "a", addr, "bind proxy address")
	flag.Parse()

	// setup orders
	if markers == "" {
		markersOrders = newAppOrder(nil)
	} else {
		var orders, err = parseOrder(markers)
		if err != nil {
			log.Fatal(err)
		}
		markersOrders = newAppOrder(orders)
	}

	// setup server

	// bind
	s.Addr = addr
	// handle all methods excluding CONNECT (no HTTPS support)
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			http.Error(w, "tunneling not supported, use plain HTTP", 405)
			return
		}
		handle(w, r, markers, filter)
	})
	// disable HTTP/2
	s.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))

	println("START ON", addr)

	// start the proxy
	log.Fatal(s.ListenAndServe())
}

// ========================================================================== //
//                                    note                                    //
// ========================================================================== //

// -------------------------------------------------------------------------- //

// don't send file meta
//
//     go run main.go -f uploadMeta

// don't send file content
//
//     go run main.go -f uploadFile

// send upload/download/delete but deliver markers in dm-wm-rm order
//
//     go run main.go -m dm-wm-rm

// -------------------------------------------------------------------------- //

// <generate random.bin>
// rm -f got.bin
// HTTP_PROXY="http://0.0.0.0:15211" ./zboxcli/zbox upload --remotepath=/remote/remote.bin --allocation "$(cat ~/.zcn/allocation.txt)" --localpath=random.bin
// HTTP_PROXY="http://0.0.0.0:15211" ./zboxcli/zbox delete --allocation "$(cat ~/.zcn/allocation.txt)" --remotepath=/remote/remote.bin
// HTTP_PROXY="http://0.0.0.0:15211" ./zboxcli/zbox download --remotepath=/remote/remote.bin --allocation "$(cat ~/.zcn/allocation.txt)" --localpath=got.bin
