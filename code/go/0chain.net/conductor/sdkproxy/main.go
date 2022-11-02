package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const (
	UPLOAD   = "/v1/file/upload/"
	COMMIT   = "/v1/connection/commit/"
	DOWNLOAD = "/v1/file/download/"
)

func init() {
	log.SetFlags(log.Ltime)
	log.SetPrefix("[SDK PROXY] ")
}

var markersOrders *appOrders

type sentRelease struct {
	once    bool
	sent    chan struct{}
	release chan struct{}
}

func newSentRelease() (sr *sentRelease) {
	sr = new(sentRelease)
	sr.once = false
	sr.sent = make(chan struct{})
	sr.release = make(chan struct{})
	return
}

type waitOrder struct {
	wait map[string]*sentRelease
}

func newWaitOrder() (wo *waitOrder) {
	wo = new(waitOrder)
	wo.wait = make(map[string]*sentRelease)
	return
}

// run in goroutine
func (wo *waitOrder) perform(orders []string) {
	log.Println("[DBG] perform", orders)
	for _, marker := range orders {
		var sr = wo.wait[marker]
		close(sr.release)
		log.Println("[DBG] release", marker)
		if marker == "rm" {
			if !sr.once {
				<-sr.sent      // wait maker sent report and release next
				sr.once = true //
				// drain
				go func(sent chan struct{}) {
					for range sent {
					}
				}(sr.sent)
			}
		} else {
			<-sr.sent // wait maker sent report and release next
		}
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
	log.Println("[DBG] add blobber", host)
	wo = newWaitOrder()
	ao.blobbers[host] = wo
	return // nil
}

func (ao *appOrders) drain() {
	for range ao.draining {
	}
}

func (ao *appOrders) addOrder(host, marker string) (sent, rel chan struct{}) {
	log.Println("[DBG] add order", host, marker)

	if len(ao.orders) == 0 {
		log.Println("[DBG] add order", host, marker, "(no orders)")
		return ao.draining, ao.unlocked // released
	}

	ao.Lock()
	defer ao.Unlock()

	var (
		wo = ao.addBlobber(host)
		sr *sentRelease
		ok bool
	)

	if sr, ok = wo.wait[marker]; ok {
		log.Println("[DBG] add order", host, marker, "(already have)")
		return sr.sent, sr.release
	}

	log.Println("[DBG] add order", host, marker, "(add)")

	rel = make(chan struct{})
	sr = newSentRelease()
	sent, rel = sr.sent, sr.release
	wo.wait[marker] = sr

	if len(wo.wait) == len(ao.orders) {
		log.Println("add order", host, marker, "PERMORM")
		go wo.perform(ao.orders) // have all required request in the queue
		// delete(ao.blobbers, host) // and remove the reference
	}

	return
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

	if err := r.ParseMultipartForm(0); err != nil {
		return nil, err
	}

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
			log.Println("-", name, ":", fh.Filename, fh.Size)
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
			log.Println("-", name, ":", val)
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
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("http write failed: %v\n", err)
	}
}

func handle(w http.ResponseWriter, r *http.Request, markers, filter string) {
	if strings.HasPrefix(r.URL.Host, "198.18.0.98") || (!strings.HasPrefix(r.URL.Host, "localhost") && !strings.HasPrefix(r.URL.Host, "198.18.0.9")) {
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

		var (
			sent, release chan struct{}
			marker        string
		)
		switch {
		case hasField(r, "write_marker"):
			if isDeleteMarker(r) {
				marker = "dm"
				log.Println("[DBG] for dm")
				sent, release = markersOrders.addOrder(r.URL.Host, "dm")
			} else {
				marker = "wm"
				log.Println("[DBG] for wm")
				sent, release = markersOrders.addOrder(r.URL.Host, "wm")
			}
		case hasField(r, "read_marker"):
			marker = "rm"
			log.Println("[DBG] for rm")
			sent, release = markersOrders.addOrder(r.URL.Host, "rm")
		}

		// send on release
		log.Println("[DBG] wait for order...")
		defer func(sent chan struct{}, marker string) {
			if marker == "rm" {
				select {
				case sent <- struct{}{}:
				default:
				}
			} else {
				sent <- struct{}{}
			}
		}(sent, marker)
		<-release
		log.Println("[DBG] released")

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

type Run []string

func (r Run) String() string {
	return fmt.Sprint([]string(r))
}

func (r *Run) Set(val string) (err error) {
	(*r) = append((*r), val)
	return
}

func waitSigInt() {
	var c = make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Printf("got signal %s, exiting...", <-c)
}

func execute(r, address string, codes chan int) {
	var (
		cmd  = exec.Command("sh", "-x", r)
		err  error
		code int
	)

	log.Print("execute: ", r)
	defer func() { log.Printf("executed (%s) with %d exit code", r, code) }()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "HTTP_PROXY=http://"+address)

	err = cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		log.Printf("executing %s: %v", r, err)
	}

	codes <- code
}

func main() {

	// address
	var (
		markers string = ""              // markers arriving order
		filter  string = ""              // filter multipart forms fields
		addr    string = "0.0.0.0:15211" // bind

		back = context.Background() //

		s   http.Server // server instance
		run Run         // run parallel with HTTP_PROXY
	)

	flag.StringVar(&markers, "m", markers, "markers arriving order")
	flag.StringVar(&filter, "f", filter, "filter multipart form fields")
	flag.StringVar(&addr, "a", addr, "bind proxy address")
	flag.Var(&run, "run", "run sh scripts parallel with HTTP_PROXY and exit then")
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

	log.Println("[INF] start on", addr)

	// start the proxy
	go func() { log.Fatal(s.ListenAndServe()) }()
	defer func() {
		if err := s.Shutdown(back); err != nil {
			log.Printf("shutdown error: %\ns", err)
		}
	}()

	if len(run) == 0 {
		waitSigInt()
		return
	}

	var codes = make(chan int, len(run))
	for _, r := range run {
		go execute(r, addr, codes)
	}

	var code int
	for i := 0; i < len(run); i++ {
		var x = <-codes
		if code == 0 && x != 0 {
			code = x
		}
	}

	os.Exit(code)
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
