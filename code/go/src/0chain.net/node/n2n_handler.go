package node

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	. "0chain.net/logging"
	"github.com/golang/snappy"
	"go.uber.org/zap"
)

/*SENDER - key used to get the connection object from the context */
const SENDER common.ContextKey = "node.sender"

const MAX_NP_REQUESTS = 2

const (
	CODEC_JSON    = 0
	CODEC_MSGPACK = 1
)

/*WithNode takes a context and adds a connection value to it */
func WithNode(ctx context.Context, node *Node) context.Context {
	return context.WithValue(ctx, SENDER, node)
}

/*GetSender returns a connection stored in the context which got created via WithConnection */
func GetSender(ctx context.Context) *Node {
	return ctx.Value(SENDER).(*Node)
}

var (
	HeaderRequestTimeStamp      = "X-Request-Timestamp"
	HeaderRequestHashData       = "X-Request-Hashdata"
	HeaderRequestHash           = "X-Request-Hash"
	HeaderRequestRelayLength    = "X-Request-Relay-Length"
	HeaderRequestMaxRelayLength = "X-Request-Max-Relay-Length"
	HeaderRequestEntityName     = "X-Request-Entity-Name"
	HeaderRequestEntityID       = "X-Request-Entity-ID"
	HeaderRequestChainID        = "X-Chain-Id"
	HeaderRequestCODEC          = "X-Chain-CODEC"

	HeaderInitialNodeID        = "X-Initial-Node-Id"
	HeaderNodeID               = "X-Node-Id"
	HeaderNodeRequestSignature = "X-Node-Request-Signature"
)

/*SendHandler is used to send any message to a given node */
type SendHandler func(n *Node) bool

/*EntitySendHandler is used to send an entity to a given node */
type EntitySendHandler func(entity datastore.Entity) SendHandler

type ReceiveEntityHandlerF func(ctx context.Context, entity datastore.Entity) (interface{}, error)

/*SendAll - send to every node */
func (np *Pool) SendAll(handler SendHandler) []*Node {
	return np.SendAtleast(len(np.Nodes), handler)
}

/*SendTo - send to a specific node */
func (np *Pool) SendTo(handler SendHandler, to string) (bool, error) {
	recepient := np.GetNode(to)
	if recepient == nil {
		return false, ErrNodeNotFound
	}
	return handler(recepient), nil
}

/*SendAtleast - It tries to communicate to at least the given number of active nodes
* TODO: May need to pass a context object so we can cancel at will.
 */
func (np *Pool) SendAtleast(numNodes int, handler SendHandler) []*Node {
	const THRESHOLD = 2
	nodes := np.shuffleNodes()
	sentTo := make([]*Node, 0, numNodes)

	if numNodes == 1 {
		node := np.sendOne(handler, nodes)
		if node == nil {
			return sentTo
		}
		sentTo = append(sentTo, node)
		return sentTo
	}
	start := time.Now()
	validCount := 0
	activeCount := 0
	numWorkers := numNodes
	if numWorkers > MAX_NP_REQUESTS {
		numWorkers = MAX_NP_REQUESTS
	}
	sendBucket := make(chan *Node, numNodes)
	validBucket := make(chan *Node, numNodes)
	done := make(chan bool)
	for i := 0; i < numWorkers; i++ {
		go func() {
			for node := range sendBucket {
				valid := handler(node)
				if valid {
					validBucket <- node
				}
				done <- true
			}
		}()
	}
	for _, node := range nodes {
		if node == Self.Node {
			continue
		}
		if node.Status == NodeStatusInactive {
			continue
		}
		sendBucket <- node
		activeCount++
	}
	doneCount := 0
	for true {
		select {
		case node := <-validBucket:
			sentTo = append(sentTo, node)
			validCount++
			if validCount == numNodes {
				close(sendBucket)
				Logger.Debug("", zap.Any("all_nodes", len(nodes)), zap.Any("requested", numNodes), zap.Any("active", activeCount), zap.Any("sent_to", len(nodes)), zap.Any("time", time.Since(start)))
				return sentTo
			}
		case <-done:
			doneCount++
			if doneCount >= numNodes+THRESHOLD || doneCount >= activeCount {
				Logger.Debug("", zap.Any("all_nodes", len(nodes)), zap.Any("requested", numNodes), zap.Any("active", activeCount), zap.Any("sent_to", len(nodes)), zap.Any("time", time.Since(start)))
				close(sendBucket)
				return sentTo
			}
		}
	}
	return sentTo
}

/*SendOne - send message to a single node in the pool */
func (np *Pool) SendOne(handler SendHandler) *Node {
	nodes := np.shuffleNodes()
	return np.sendOne(handler, nodes)
}

func (np *Pool) sendOne(handler SendHandler, nodes []*Node) *Node {
	for _, node := range nodes {
		if node.Status == NodeStatusInactive {
			continue
		}
		valid := handler(node)
		if valid {
			return node
		}
	}
	return nil
}

/*SetHeaders - sets the request headers
 */
func SetHeaders(req *http.Request, entity datastore.Entity, options *SendOptions) bool {
	ts := common.Now()
	hashdata := fmt.Sprintf("%v:%v:%v", Self.GetKey(), ts, entity.GetKey())
	hash := encryption.Hash(hashdata)
	signature, err := Self.Sign(hash)
	if err != nil {
		return false
	}
	req.Header.Set(HeaderRequestChainID, config.GetServerChainID())
	req.Header.Set(HeaderNodeID, Self.GetKey())
	if options.InitialNodeID != "" {
		req.Header.Set(HeaderInitialNodeID, options.InitialNodeID)
	}
	req.Header.Set(HeaderRequestTimeStamp, strconv.FormatInt(int64(ts), 10))
	req.Header.Set(HeaderRequestHashData, hashdata)
	req.Header.Set(HeaderRequestHash, hash)
	req.Header.Set(HeaderNodeRequestSignature, signature)
	req.Header.Set(HeaderRequestEntityName, entity.GetEntityMetadata().GetName())
	req.Header.Set(HeaderRequestEntityID, datastore.ToString(entity.GetKey()))
	if options.CODEC == 0 {
		req.Header.Set(HeaderRequestCODEC, "JSON")
	} else {
		req.Header.Set(HeaderRequestCODEC, "Msgpack")
	}
	if options.MaxRelayLength > 0 {
		req.Header.Set(HeaderRequestMaxRelayLength, strconv.FormatInt(options.MaxRelayLength, 10))
	}
	req.Header.Set(HeaderRequestRelayLength, strconv.FormatInt(options.CurrentRelayLength, 10))
	return true
}

/*SendOptions - options to tune how the messages are sent within the network */
type SendOptions struct {
	Timeout            time.Duration
	MaxRelayLength     int64
	CurrentRelayLength int64
	Compress           bool
	InitialNodeID      string
	CODEC              int
}

/*SendEntityHandler provides a client API to send an entity */
func SendEntityHandler(uri string, options *SendOptions) EntitySendHandler {
	return func(entity datastore.Entity) SendHandler {
		return func(n *Node) bool {
			url := fmt.Sprintf("%v%v", n.GetURLBase(), uri)
			timeout := 500 * time.Millisecond
			if options.Timeout > 0 {
				timeout = options.Timeout
			}
			client := &http.Client{Timeout: timeout}

			var buffer *bytes.Buffer
			if options.CODEC == datastore.CodecJSON {
				buffer = datastore.ToJSON(entity)
			} else {
				buffer = datastore.ToMsgpack(entity)
			}
			if options.Compress {
				cbytes := snappy.Encode(nil, buffer.Bytes())
				buffer = bytes.NewBuffer(cbytes)
			}
			req, err := http.NewRequest("POST", url, buffer)
			if err != nil {
				return false
			}
			defer req.Body.Close()

			if options.Compress {
				req.Header.Set("Content-Encoding", "snappy")
			}
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			SetHeaders(req, entity, options)
			delay := common.InduceDelay()
			Logger.Debug("sending", zap.Any("from", Self.SetIndex), zap.Any("to", n.SetIndex), zap.Any("handler", url), zap.Any("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Any("delay", delay))
			resp, err := client.Do(req)
			if err != nil {
				Logger.Error("sending", zap.Any("from", Self.SetIndex), zap.Any("to", n.SetIndex), zap.Any("handler", url), zap.Any("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Error(err))
				return false
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				var rbuf bytes.Buffer
				rbuf.ReadFrom(resp.Body)
				Logger.Error("sending", zap.Any("from", Self.SetIndex), zap.Any("to", n.SetIndex), zap.Any("handler", url), zap.Any("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Any("status_code", resp.StatusCode), zap.Any("response", rbuf.String()))
				return false
			}
			io.Copy(ioutil.Discard, resp.Body)
			return true
		}
	}
}

/*ToN2NReceiveEntityHandler - takes a handler that accepts an entity, processes and responds and converts it
* into somethign suitable for Node 2 Node communication*/
func ToN2NReceiveEntityHandler(handler datastore.JSONEntityReqResponderF) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-type")
		if !strings.HasPrefix(contentType, "application/json") {
			http.Error(w, "Header Content-type=application/json not found", 400)
			return
		}

		chainID := r.Header.Get(HeaderRequestChainID)
		if config.GetServerChainID() != chainID {
			//TODO: We can't do this in cross-chain messaging
			return
		}

		nodeID := r.Header.Get(HeaderNodeID)
		sender := GetNode(nodeID)
		if sender == nil {
			Logger.Error("message received", zap.Any("from", nodeID), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "request from unrecognized node"))
			return
		}

		entityName := r.Header.Get(HeaderRequestEntityName)
		if entityName == "" {
			Logger.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "entity name blank"))
			return
		}

		reqHashdata := r.Header.Get(HeaderRequestHashData)
		reqHash := r.Header.Get(HeaderRequestHash)
		if reqHash != encryption.Hash(reqHashdata) {
			Logger.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "request data hash invalid"))
			return
		}
		reqSignature := r.Header.Get(HeaderNodeRequestSignature)
		if ok, _ := sender.Verify(reqSignature, reqHash); !ok {
			return
		}

		Logger.Debug("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("data", reqHashdata))
		reqTS := r.Header.Get(HeaderRequestTimeStamp)
		if reqTS == "" {
			Logger.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("data", reqHashdata), zap.Any("error", "no timestamp for the message"))
			return
		}
		reqTSn, err := strconv.ParseInt(reqTS, 10, 64)
		if err != nil {
			Logger.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("data", reqHashdata), zap.Error(err))
			return
		}
		if !common.Within(reqTSn, 5) {

		}

		entityMetadata := datastore.GetEntityMetadata(entityName)
		if entityMetadata == nil {
			return
		}
		var buffer io.Reader = r.Body
		defer r.Body.Close()
		if r.Header.Get("Content-Encoding") == "snappy" {
			cbuffer := new(bytes.Buffer)
			cbuffer.ReadFrom(r.Body)

			cbytes, err := snappy.Decode(nil, cbuffer.Bytes())
			if err != nil {
				Logger.Error("snappy decoding", zap.Any("error", err))
				return
			}
			buffer = bytes.NewReader(cbytes)
		}
		entity := entityMetadata.Instance()

		if r.Header.Get(HeaderRequestCODEC) == "JSON" {
			err = datastore.FromJSON(buffer, entity.(datastore.Entity))
			if err != nil {
				Logger.Error("json decoding", zap.Any("error", err))
			}
		} else {
			err = datastore.FromMsgpack(buffer, entity.(datastore.Entity))
			if err != nil {
				Logger.Error("msgpack decoding", zap.Any("error", err))
			}
		}

		if err != nil {
			http.Error(w, "Error decoding json", 500)
			return
		}

		entity.ComputeProperties()
		ctx := r.Context()
		initialNodeID := r.Header.Get(HeaderInitialNodeID)
		if initialNodeID != "" {
			initSender := GetNode(initialNodeID)
			if initSender == nil {
				return
			}
			ctx = WithNode(ctx, initSender)
		} else {
			ctx = WithNode(ctx, sender)
		}
		delay := common.InduceDelay()
		if delay > 0 {
			Logger.Debug("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("data", reqHashdata), zap.Any("delay", delay))
		}
		data, err := handler(ctx, entity)
		common.Respond(w, data, err)
	}
}

/*SetupN2NHandlers - Setup all the node 2 node communiations*/
func SetupN2NHandlers() {
	http.HandleFunc("/v1/_n2n/entity/post", ToN2NReceiveEntityHandler(datastore.PrintEntityHandler))
}
