package node

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"time"

	"0chain.net/common"
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

/*SendOne - send message to a single node in the pool */
func (np *Pool) SendOne(handler SendHandler) *Node {
	nodes := np.shuffleNodes()
	return np.sendOne(handler, nodes)
}

/*SendToMultiple - send to multiple nodes */
func (np *Pool) SendToMultiple(handler SendHandler, nodes []*Node) (bool, error) {
	sentTo := np.sendTo(len(nodes), nodes, handler)
	if len(sentTo) == len(nodes) {
		return true, nil
	}
	return false, common.NewError("send_to_given_nodes_unsuccessful", "Sending to given nodes not successful")
}

/*SendAtleast - It tries to communicate to at least the given number of active nodes */
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
				N2n.Info("send message (valid)", zap.Int("all_nodes", len(nodes)), zap.Int("requested", numNodes), zap.Int("active", activeCount), zap.Int("sent_to", len(sentTo)), zap.Duration("time", time.Since(start)))
				return sentTo
			}
		case <-done:
			doneCount++
			if doneCount >= numNodes+THRESHOLD || doneCount >= activeCount {
				N2n.Info("send message (done)", zap.Int("all_nodes", len(nodes)), zap.Int("requested", numNodes), zap.Int("active", activeCount), zap.Int("sent_to", len(sentTo)), zap.Duration("time", time.Since(start)))
				close(sendBucket)
				return sentTo
			}
		}
	}
	return sentTo
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

var n2nTrace = &httptrace.ClientTrace{}

func init() {
	n2nTrace.GotConn = func(connInfo httptrace.GotConnInfo) {
		fmt.Printf("GOT conn: %+v\n", connInfo)
	}
}

/*SendEntityHandler provides a client API to send an entity */
func SendEntityHandler(uri string, options *SendOptions) EntitySendHandler {
	timeout := 500 * time.Millisecond
	if options.Timeout > 0 {
		timeout = options.Timeout
	}
	return func(entity datastore.Entity) SendHandler {
		data := getResponseData(options, entity).Bytes()
		toPull := false
		if len(data) > LargeMessageThreshold {
			toPull = true
			key := p2pKey(uri, entity.GetKey())
			pdce := &PushDataCacheEntry{Options: *options, Data: data, EntityName: entity.GetEntityMetadata().GetName()}
			pushDataCache.Add(key, pdce)
		}
		return func(receiver *Node) bool {
			timer := receiver.GetTimer(uri)
			url := receiver.GetN2NURLBase() + uri
			var buffer *bytes.Buffer
			push := true
			if toPull {
				pushTime := timer.Mean()
				pullTimer := receiver.GetTimer(serveMetricKey(uri))
				pullTime := receiver.SmallMessageSendTime
				if pullTimer != nil {
					pullTime = pullTimer.Mean()
				}
				if pushTime > pullTime+2*receiver.SmallMessageSendTime {
					N2n.Info("sending - push to pull", zap.Int("from", Self.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.String("id", entity.GetKey()))
					buffer = bytes.NewBuffer(nil)
					push = false
				}
			}
			if push {
				buffer = bytes.NewBuffer(data)
			}
			req, err := http.NewRequest("POST", url, buffer)
			if err != nil {
				return false
			}
			defer req.Body.Close()

			if options.Compress {
				req.Header.Set("Content-Encoding", compDecomp.Encoding())
			}
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			delay := common.InduceDelay()
			N2n.Debug("sending", zap.Int("from", Self.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.String("id", entity.GetKey()), zap.Any("delay", delay))
			SetSendHeaders(req, entity, options)
			ctx, cancel := context.WithCancel(context.TODO())
			req = req.WithContext(ctx)
			// Keep the number of messages to a node bounded
			receiver.Grab()
			time.AfterFunc(timeout, cancel)
			ts := time.Now()
			Self.Node.LastActiveTime = ts
			//req = req.WithContext(httptrace.WithClientTrace(req.Context(), n2nTrace))
			resp, err := httpClient.Do(req)
			receiver.Release()
			if push {
				timer.UpdateSince(ts)
				sizer := receiver.GetSizeMetric(uri)
				sizer.Update(int64(len(data)))
			}
			N2n.Info("sending", zap.Int("from", Self.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()))

			if err != nil {
				receiver.SendErrors++
				N2n.Error("sending", zap.Int("from", Self.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Error(err))
				return false
			}
			defer resp.Body.Close()
			if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent) {
				var rbuf bytes.Buffer
				rbuf.ReadFrom(resp.Body)
				N2n.Error("sending", zap.Int("from", Self.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Any("status_code", resp.StatusCode), zap.String("response", rbuf.String()))
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

func validateSendRequest(sender *Node, r *http.Request) bool {
	entityName := r.Header.Get(HeaderRequestEntityName)
	entityID := r.Header.Get(HeaderRequestEntityID)
	N2n.Debug("message received", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName))
	if !validateChain(sender, r) {
		N2n.Error("message received - invalid chain", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName))
		return false
	}
	if !validateEntityMetadata(sender, r) {
		N2n.Error("message received - invalid entity metadata", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName))
		return false
	}
	if entityID == "" {
		N2n.Error("message received - entity id blank", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI))
		return false
	}
	reqTS := r.Header.Get(HeaderRequestTimeStamp)
	if reqTS == "" {
		N2n.Error("message received - no timestamp for the message", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName), zap.Any("id", entityID))
		return false
	}
	reqTSn, err := strconv.ParseInt(reqTS, 10, 64)
	if err != nil {
		N2n.Error("message received", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName), zap.Any("id", entityID), zap.Error(err))
		return false
	}
	sender.Status = NodeStatusActive
	sender.LastActiveTime = time.Unix(reqTSn, 0)
	Self.Node.LastActiveTime = time.Now()
	if !common.Within(reqTSn, N2NTimeTolerance) {
		N2n.Error("message received - tolerance", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("enitty", entityName), zap.Int64("ts", reqTSn), zap.Time("tstime", time.Unix(reqTSn, 0)))
		return false
	}

	reqHashdata := getHashData(sender.GetKey(), common.Timestamp(reqTSn), entityID)
	reqHash := r.Header.Get(HeaderRequestHash)
	if reqHash != encryption.Hash(reqHashdata) {
		N2n.Error("message received - request data hash invalid", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("hash", reqHash), zap.String("hashdata", reqHashdata))
		return false
	}
	reqSignature := r.Header.Get(HeaderNodeRequestSignature)
	if ok, _ := sender.Verify(reqSignature, reqHash); !ok {
		N2n.Error("message received - invalid signature", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("hash", reqHash), zap.String("hashdata", reqHashdata), zap.String("signature", reqSignature))
		return false
	}

	sender.Status = NodeStatusActive
	sender.LastActiveTime = time.Unix(reqTSn, 0)

	return true
}

/*ToN2NReceiveEntityHandler - takes a handler that accepts an entity, processes and responds and converts it
* into somethign suitable for Node 2 Node communication*/
func ToN2NReceiveEntityHandler(handler datastore.JSONEntityReqResponderF, options *ReceiveOptions) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-type")
		if !strings.HasPrefix(contentType, "application/json") {
			http.Error(w, "Header Content-type=application/json not found", 400)
			return
		}
		nodeID := r.Header.Get(HeaderNodeID)
		sender := GetNode(nodeID)
		if sender == nil {
			N2n.Error("message received - request from unrecognized node", zap.String("from", nodeID), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI))
			return
		}
		if !validateSendRequest(sender, r) {
			return
		}
		entityName := r.Header.Get(HeaderRequestEntityName)
		entityID := r.Header.Get(HeaderRequestEntityID)
		entityMetadata := datastore.GetEntityMetadata(entityName)
		if options != nil && options.MessageFilter != nil {
			if !options.MessageFilter.Accept(entityName, entityID) {
				defer r.Body.Close()
				io.Copy(ioutil.Discard, r.Body)
				N2n.Debug("message receive - reject", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity_id", entityID))
				return
			}
		}
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
		entity, err := getRequestEntity(r, entityMetadata)
		if err != nil {
			if err == NoDataErr {
				go pullEntityHandler(ctx, sender, r.RequestURI, handler, entityName, entityID)
				sender.Received++
				return
			}
			http.Error(w, fmt.Sprintf("Error reading entity: %v", err), 500)
			return
		}
		if entity.GetKey() != entityID {
			N2n.Error("message received - entity id doesn't match with signed id", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity_id", entityID), zap.String("entity.id", entity.GetKey()))
			return
		}
		entity.ComputeProperties()
		delay := common.InduceDelay()
		if delay > 0 {
			N2n.Debug("message received", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName), zap.Any("id", entityID), zap.Any("delay", delay))
		}
		start := time.Now()
		data, err := handler(ctx, entity)
		duration := time.Since(start)
		common.Respond(w, data, err)
		if err != nil {
			N2n.Error("message received", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()), zap.Error(err))
		} else {
			N2n.Info("message received", zap.Int("from", sender.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", r.RequestURI), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()))
		}
		sender.Received++
	}
}

func p2pKey(uri string, id string) string {
	return uri + ":" + id
}

//PushToPullHandler - handles a pull request of cached push entity data
func PushToPullHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	pushURI := r.FormValue("_puri")
	id := r.FormValue("id")
	key := p2pKey(pushURI, id)
	pcde, err := pushDataCache.Get(key)
	if err != nil {
		N2n.Error("push to pull", zap.String("key", key), zap.Error(err))
		return nil, common.NewError("request_data_not_found", "Requested data is not found")
	}
	N2n.Info("push to pull", zap.String("key", key))
	return pcde, nil
}

/*pullEntityHandler - pull an entity that wasn't pushed as it's large and pulling is cheaper */
func pullEntityHandler(ctx context.Context, nd *Node, uri string, handler datastore.JSONEntityReqResponderF, entityName string, entityID datastore.Key) {
	phandler := func(pctx context.Context, entity datastore.Entity) (interface{}, error) {
		Logger.Info("pull entity", zap.String("entity", entityName), zap.Any("id", entityID))
		if entity.GetEntityMetadata().GetName() != entityName {
			return entity, nil
		}
		if entity.GetKey() != entityID {
			return entity, nil
		}
		start := time.Now()
		_, err := handler(ctx, entity)
		duration := time.Since(start)
		if err != nil {
			N2n.Error("message pull", zap.Int("from", nd.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", uri), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()), zap.Error(err))
		} else {
			N2n.Info("message pull", zap.Int("from", nd.SetIndex), zap.Int("to", Self.SetIndex), zap.String("handler", uri), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()))
		}
		return entity, nil
	}
	params := make(map[string]string)
	params["_puri"] = uri
	params["id"] = datastore.ToString(entityID)
	rhandler := pullDataRequestor(params, phandler)
	result := rhandler(nd)
	N2n.Info("message pull", zap.String("uri", uri), zap.String("entity", entityName), zap.String("id", entityID), zap.Bool("result", result))
}

var pullDataRequestor EntityRequestor

func init() {
	http.HandleFunc("/v1/n2n/entity_pull/get", ToN2NSendEntityHandler(PushToPullHandler))

	options := &SendOptions{Timeout: TimeoutLargeMessage, CODEC: CODEC_MSGPACK, Compress: true}
	pullDataRequestor = RequestEntityHandler("/v1/n2n/entity_pull/get", options, nil)
}
