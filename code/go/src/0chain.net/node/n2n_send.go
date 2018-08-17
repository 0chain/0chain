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
	"go.uber.org/zap"
)

/*MaxConcurrentRequests - max number of concurrent requests when sending a message to the node pool */
var MaxConcurrentRequests = 2

/*SetMaxConcurrentRequests - set the max number of concurrent requests */
func SetMaxConcurrentRequests(maxConcurrentRequests int) {
	MaxConcurrentRequests = maxConcurrentRequests
}

/*EntitySendHandler is used to send an entity to a given node */
type EntitySendHandler func(entity datastore.Entity) SendHandler

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

/*SendToMultiple - send to multiple nodes */
func (np *Pool) SendToMultiple(handler SendHandler, nodes []*Node) (bool, error) {
	sentTo := np.sendTo(len(nodes), nodes, handler)
	if len(sentTo) == len(nodes) {
		return true, nil
	}
	return false, common.NewError("send_to_given_nodes_unsuccessful", "Sending to given nodes not successful")
}

/*SendAtleast - It tries to communicate to at least the given number of active nodes
* TODO: May need to pass a context object so we can cancel at will.
 */
func (np *Pool) SendAtleast(numNodes int, handler SendHandler) []*Node {
	nodes := np.shuffleNodes()
	return np.sendTo(numNodes, nodes, handler)
}

func (np *Pool) sendTo(numNodes int, nodes []*Node, handler SendHandler) []*Node {
	const THRESHOLD = 2
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
	if numWorkers > MaxConcurrentRequests && MaxConcurrentRequests > 0 {
		numWorkers = MaxConcurrentRequests
	}
	sendBucket := make(chan *Node, numNodes)
	validBucket := make(chan *Node, numNodes)
	done := make(chan bool, numNodes)
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
	if activeCount == 0 {
		Logger.Debug("send message (no active nodes)")
		close(sendBucket)
		return sentTo
	}
	doneCount := 0
	for true {
		select {
		case node := <-validBucket:
			sentTo = append(sentTo, node)
			validCount++
			if validCount == numNodes {
				close(sendBucket)
				N2n.Info("send message (valid)", zap.Any("all_nodes", len(nodes)), zap.Any("requested", numNodes), zap.Any("active", activeCount), zap.Any("sent_to", len(sentTo)), zap.Any("time", time.Since(start)))
				return sentTo
			}
		case <-done:
			doneCount++
			if doneCount >= numNodes+THRESHOLD || doneCount >= activeCount {
				N2n.Info("send message (done)", zap.Any("all_nodes", len(nodes)), zap.Any("requested", numNodes), zap.Any("active", activeCount), zap.Any("sent_to", len(sentTo)), zap.Any("time", time.Since(start)))
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

/*SendEntityHandler provides a client API to send an entity */
func SendEntityHandler(uri string, options *SendOptions) EntitySendHandler {
	return func(entity datastore.Entity) SendHandler {
		return func(receiver *Node) bool {
			timer := receiver.GetTimer(uri)
			timeout := 500 * time.Millisecond
			if options.Timeout > 0 {
				timeout = options.Timeout
			}
			buffer := getResponseData(options, entity)
			url := receiver.GetN2NURLBase() + uri
			req, err := http.NewRequest("POST", url, buffer)
			if err != nil {
				return false
			}
			defer req.Body.Close()

			if options.Compress {
				req.Header.Set("Content-Encoding", "snappy")
			}
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			delay := common.InduceDelay()
			N2n.Debug("sending", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Any("handler", uri), zap.Any("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Any("delay", delay))
			SetSendHeaders(req, entity, options)
			ctx, cancel := context.WithCancel(context.TODO())
			req = req.WithContext(ctx)
			time.AfterFunc(timeout, cancel)
			// Keep the number of messages to a node bounded
			receiver.Grab()
			ts := time.Now()
			resp, err := httpClient.Do(req)
			receiver.Release()
			timer.UpdateSince(ts)
			N2n.Info("sending", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Duration("duration", time.Since(ts)), zap.Any("handler", uri), zap.Any("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()))

			if err != nil {
				N2n.Error("sending", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Duration("duration", time.Since(ts)), zap.Any("handler", uri), zap.Any("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Error(err))
				return false
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				var rbuf bytes.Buffer
				rbuf.ReadFrom(resp.Body)
				N2n.Error("sending", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Duration("duration", time.Since(ts)), zap.Any("handler", uri), zap.Any("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Any("status_code", resp.StatusCode), zap.Any("response", rbuf.String()))
				return false
			}
			receiver.Status = NodeStatusActive
			receiver.LastActiveTime = time.Now()
			io.Copy(ioutil.Discard, resp.Body)
			return true
		}
	}
}

/*SetSendHeaders - sets the send request headers*/
func SetSendHeaders(req *http.Request, entity datastore.Entity, options *SendOptions) bool {
	SetHeaders(req)
	if options.InitialNodeID != "" {
		req.Header.Set(HeaderInitialNodeID, options.InitialNodeID)
	}
	req.Header.Set(HeaderRequestEntityName, entity.GetEntityMetadata().GetName())
	req.Header.Set(HeaderRequestEntityID, datastore.ToString(entity.GetKey()))
	ts := common.Now()
	hashdata := getHashData(Self.GetKey(), ts, entity.GetKey())
	hash := encryption.Hash(hashdata)
	signature, err := Self.Sign(hash)
	if err != nil {
		return false
	}
	req.Header.Set(HeaderRequestTimeStamp, strconv.FormatInt(int64(ts), 10))
	req.Header.Set(HeaderRequestHash, hash)
	req.Header.Set(HeaderNodeRequestSignature, signature)

	if options.CODEC == 0 {
		req.Header.Set(HeaderRequestCODEC, CodecJSON)
	} else {
		req.Header.Set(HeaderRequestCODEC, CodecMsgpack)
	}
	if options.MaxRelayLength > 0 {
		req.Header.Set(HeaderRequestMaxRelayLength, strconv.FormatInt(options.MaxRelayLength, 10))
	}
	req.Header.Set(HeaderRequestRelayLength, strconv.FormatInt(options.CurrentRelayLength, 10))
	return true
}

func validateChain(sender *Node, r *http.Request) bool {
	chainID := r.Header.Get(HeaderRequestChainID)
	if config.GetServerChainID() != chainID {
		return false
	}
	return true
}

func validateEntityMetadata(sender *Node, r *http.Request) bool {
	entityName := r.Header.Get(HeaderRequestEntityName)
	if entityName == "" {
		N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "entity name blank"))
		return false
	}
	entityMetadata := datastore.GetEntityMetadata(entityName)
	if entityMetadata == nil {
		N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("error", "unknown entity"))
		return false
	}
	return true
}

func validateSendRequest(sender *Node, r *http.Request) bool {
	if !validateChain(sender, r) {
		return false
	}
	if !validateEntityMetadata(sender, r) {
		return false
	}
	entityName := r.Header.Get(HeaderRequestEntityName)
	entityID := r.Header.Get(HeaderRequestEntityID)
	if entityID == "" {
		N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "entity id blank"))
		return false
	}
	N2n.Debug("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName))
	reqTS := r.Header.Get(HeaderRequestTimeStamp)
	if reqTS == "" {
		N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("id", entityID), zap.Any("error", "no timestamp for the message"))
		return false
	}
	reqTSn, err := strconv.ParseInt(reqTS, 10, 64)
	if err != nil {
		N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("id", entityID), zap.Error(err))
		return false
	}

	if !common.Within(reqTSn, N2NTimeTolerance) {
		N2n.Debug("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("enitty", entityName), zap.Any("ts", reqTSn), zap.Any("tstime", time.Unix(reqTSn, 0)))
		return false
	}

	reqHashdata := getHashData(sender.GetKey(), common.Timestamp(reqTSn), entityID)
	reqHash := r.Header.Get(HeaderRequestHash)
	if reqHash != encryption.Hash(reqHashdata) {
		N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "request data hash invalid"), zap.String("hash", reqHash), zap.String("hashdata", reqHashdata))
		return false
	}
	reqSignature := r.Header.Get(HeaderNodeRequestSignature)
	if ok, _ := sender.Verify(reqSignature, reqHash); !ok {
		return false
	}

	sender.Status = NodeStatusActive
	sender.LastActiveTime = time.Unix(reqTSn, 0)

	return true
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

		nodeID := r.Header.Get(HeaderNodeID)
		sender := GetNode(nodeID)
		if sender == nil {
			N2n.Error("message received", zap.Any("from", nodeID), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "request from unrecognized node"))
			return
		}
		if !validateSendRequest(sender, r) {
			return
		}
		entityName := r.Header.Get(HeaderRequestEntityName)
		entityMetadata := datastore.GetEntityMetadata(entityName)
		entity, err := getRequestEntity(r, entityMetadata)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading entity: %v", err), 500)
			return
		}
		entityID := r.Header.Get(HeaderRequestEntityID)
		if entity.GetKey() != entityID {
			N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.String("entity_id", entityID), zap.String("entity.id", entity.GetKey()), zap.Any("error", "entity id doesn't match with signed id"))
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
			N2n.Debug("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("id", entityID), zap.Any("delay", delay))
		}
		data, err := handler(ctx, entity)
		common.Respond(w, data, err)
		if err != nil {
			N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("id", entity.GetKey()), zap.Error(err))
		} else {
			N2n.Info("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("entity", entityName), zap.Any("id", entity.GetKey()))
		}
		sender.Received++
	}
}
