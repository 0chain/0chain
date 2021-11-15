package node

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

var ErrSendingToSelf = common.NewError("sending_to_self", "Message can't be sent to oneself")

/*MaxConcurrentRequests - max number of concurrent requests when sending a message to the node pool */
var (
	MaxConcurrentRequests        = 2
	n2nVerifyRequestsWithContext = common.NewWithContextFunc(4)
)

/*SetMaxConcurrentRequests - set the max number of concurrent requests */
func SetMaxConcurrentRequests(maxConcurrentRequests int) {
	MaxConcurrentRequests = maxConcurrentRequests
}

/*SendAll - send to every node */
func (np *Pool) SendAll(ctx context.Context, handler SendHandler) []*Node {
	ts := time.Now()
	defer func() {
		if time.Since(ts) > time.Second*3 {
			logging.Logger.Warn("Send to all slow - more than 3 seconds",
				zap.Any("duration", time.Since(ts)),
				zap.Int("num", np.Size()))
		}
	}()

	return np.SendAtleast(ctx, np.Size(), handler)
}

/*SendTo - send to a specific node */
func (np *Pool) SendTo(ctx context.Context, handler SendHandler, to string) (bool, error) {
	recepient := np.GetNode(to)
	if recepient == nil {
		return false, ErrNodeNotFound
	}
	if Self.IsEqual(recepient) {
		return false, ErrSendingToSelf
	}
	return handler(ctx, recepient), nil
}

/*SendToMultipleNodes - send to multiple nodes */
func (np *Pool) SendToMultipleNodes(ctx context.Context, handler SendHandler, nodes []*Node) (result []*Node) {
	defer func() {
		if r := recover(); r != nil {
			logging.Logger.Error("PANIC", zap.Any("error", r))
		}
	}()
	result = np.sendTo(ctx, len(nodes), nodes, handler)
	return
}

/*SendAtleast - It tries to communicate to at least the given number of active nodes */
func (np *Pool) SendAtleast(ctx context.Context, numNodes int, handler SendHandler) []*Node {
	nodes := np.shuffleNodes(false)
	return np.sendTo(ctx, numNodes, nodes, handler)
}

func (np *Pool) sendTo(ctx context.Context, numNodes int, nodes []*Node, handler SendHandler) []*Node {
	const THRESHOLD = 2
	sentTo := make([]*Node, 0, numNodes)
	if numNodes == 1 {
		node := np.sendOne(ctx, handler, nodes)
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
				if node != nil {
					if handler(ctx, node) {
						validBucket <- node
					}
				}

				done <- true
			}
		}()
	}
	for _, node := range nodes {
		if Self.IsEqual(node) {
			continue
		}
		if node.GetStatus() == NodeStatusInactive {
			continue
		}
		sendBucket <- node
		activeCount++
		if activeCount == numNodes {
			break
		}
	}

	if activeCount == 0 {
		//N2n.Debug("send message (no active nodes)")
		close(sendBucket)
		return sentTo
	}
	doneCount := 0
	for {
		select {
		case node := <-validBucket:
			sentTo = append(sentTo, node)
			validCount++
			if validCount == numNodes {
				close(sendBucket)
				logging.N2n.Info("send message (valid)", zap.Int("all_nodes", len(nodes)), zap.Int("requested", numNodes), zap.Int("active", activeCount), zap.Int("sent_to", len(sentTo)), zap.Duration("time", time.Since(start)))
				return sentTo
			}
		case <-done:
			doneCount++
			if doneCount >= numNodes+THRESHOLD || doneCount >= activeCount {
				logging.N2n.Info("send message (done)", zap.Int("all_nodes", len(nodes)), zap.Int("requested", numNodes), zap.Int("active", activeCount), zap.Int("sent_to", len(sentTo)), zap.Duration("time", time.Since(start)))
				close(sendBucket)
				return sentTo
			}
		}
	}
}

func (np *Pool) sendOne(ctx context.Context, handler SendHandler, nodes []*Node) *Node {
	for _, node := range nodes {
		if node.GetStatus() == NodeStatusInactive {
			continue
		}
		if handler(ctx, node) {
			return node
		}
	}
	return nil
}

func shouldPush(options *SendOptions, receiver *Node, uri string, entity datastore.Entity, timer metrics.Timer) bool {
	if options.Pull {
		return false
	}
	if timer.Count() < 50 {
		return true
	}
	if pullSendTimer := receiver.GetTimer(serveMetricKey(uri)); pullSendTimer != nil && pullSendTimer.Count() < 50 {
		return false
	}
	pushTime := timer.Mean()
	push2pullTime := getPushToPullTime(receiver)
	if pushTime > push2pullTime {
		return false
	}
	return true
}

/*SendEntityHandler provides a client API to send an entity */
func SendEntityHandler(uri string, options *SendOptions) EntitySendHandler {
	timeout := 500 * time.Millisecond
	if options.Timeout > 0 {
		timeout = options.Timeout
	}
	return func(entity datastore.Entity) SendHandler {
		data := getResponseData(options, entity).Bytes()

		toPull := options.Pull
		if len(data) > LargeMessageThreshold || toPull {
			toPull = true
			key := p2pKey(uri, entity.GetKey())
			pdce := &pushDataCacheEntry{Options: *options, Data: data, EntityName: entity.GetEntityMetadata().GetName()}
			pushDataCache.Add(key, pdce)
		}
		return func(ctx context.Context, receiver *Node) bool {
			timer := receiver.GetTimer(uri)
			addr := receiver.GetN2NURLBase() + uri
			var buffer *bytes.Buffer
			push := !toPull || shouldPush(options, receiver, uri, entity, timer)
			if push {
				buffer = bytes.NewBuffer(data)
			} else {
				buffer = bytes.NewBuffer(nil)
			}
			req, err := http.NewRequest("POST", addr, buffer)
			if err != nil {
				return false
			}
			defer req.Body.Close()

			if options.Compress {
				req.Header.Set("Content-Encoding", compDecomp.Encoding())
			}
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			SetSendHeaders(req, entity, options)
			// Keep the number of messages to a node bounded
			var (
				selfNode *Node
				resp     *http.Response
				ts       = time.Now()
			)

			func() {
				receiver.Grab()
				defer receiver.Release()

				selfNode = Self.Underlying()
				selfNode.SetLastActiveTime(ts)
				selfNode.InduceDelay(receiver)

				cctx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()
				req = req.WithContext(cctx)
				//req = req.WithContext(httptrace.WithClientTrace(req.Context(), n2nTrace))
				resp, err = httpClient.Do(req)
			}()

			logging.N2n.Info("sending", zap.Int("from", selfNode.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()))
			switch err {
			case nil:
			default:
				ue, ok := err.(*url.Error)
				if ok && ue.Unwrap() != context.Canceled {
					receiver.AddSendErrors(1)
					receiver.AddErrorCount(1)
					logging.N2n.Error("sending", zap.Int("from", selfNode.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Error(err))
				}
				return false
			}

			receiver.SetStatus(NodeStatusActive)
			receiver.SetLastActiveTime(time.Now())
			receiver.SetErrorCount(receiver.GetSendErrors())

			readAndClose(resp.Body)
			if push {
				timer.UpdateSince(ts)
				sizer := receiver.GetSizeMetric(uri)
				sizer.Update(int64(len(data)))
			}
			if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent) {
				logging.N2n.Error("sending", zap.Int("from", selfNode.SetIndex), zap.Int("to", receiver.SetIndex), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Any("status_code", resp.StatusCode))
				return false
			}
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
	hashdata := getHashData(Self.Underlying().GetKey(), ts, entity.GetKey())
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
	selfSetIndex := Self.Underlying().SetIndex
	if !validateChain(sender, r) {
		logging.N2n.Error("message received - invalid chain", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName))
		return false
	}
	if !validateEntityMetadata(sender, r) {
		logging.N2n.Error("message received - invalid entity metadata", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName))
		return false
	}
	if entityID == "" {
		logging.N2n.Error("message received - entity id blank", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI))
		return false
	}
	reqTS := r.Header.Get(HeaderRequestTimeStamp)
	if reqTS == "" {
		logging.N2n.Error("message received - no timestamp for the message", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName), zap.Any("id", entityID))
		return false
	}
	reqTSn, err := strconv.ParseInt(reqTS, 10, 64)
	if err != nil {
		logging.N2n.Error("message received", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI), zap.String("entity", entityName), zap.Any("id", entityID), zap.Error(err))
		return false
	}
	sender.SetStatus(NodeStatusActive)
	sender.SetLastActiveTime(time.Unix(reqTSn, 0))
	Self.Underlying().SetLastActiveTime(time.Now())
	if !common.Within(reqTSn, int64(N2NTimeTolerance*time.Second)) {
		logging.N2n.Error("message received - tolerance", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI),
			zap.String("enitty", entityName), zap.String("id", entityID),
			zap.Int64("ts", reqTSn), zap.Time("tstime", time.Unix(reqTSn, 0)))
		return false
	}

	reqHashdata := getHashData(sender.GetKey(), common.Timestamp(reqTSn), entityID)
	reqHash := r.Header.Get(HeaderRequestHash)
	if reqHash != encryption.Hash(reqHashdata) {
		logging.N2n.Error("message received - request data hash invalid", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI), zap.String("hash", reqHash), zap.String("hashdata", reqHashdata))
		return false
	}
	reqSignature := r.Header.Get(HeaderNodeRequestSignature)
	if ok, _ := sender.Verify(reqSignature, reqHash); !ok {
		logging.N2n.Error("message received - invalid signature", zap.Int("from", sender.SetIndex),
			zap.Int("to", selfSetIndex), zap.String("handler", r.RequestURI), zap.String("hash", reqHash), zap.String("hashdata", reqHashdata), zap.String("signature", reqSignature))
		return false
	}
	sender.SetStatus(NodeStatusActive)
	sender.SetLastActiveTime(time.Unix(reqTSn, 0))
	return true
}

/*ToN2NReceiveEntityHandler - takes a handler that accepts an entity, processes and responds and converts it
* into something suitable for Node 2 Node communication*/
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
			logging.N2n.Error("message received - request from unrecognized node", zap.String("from", nodeID),
				zap.Int("to", Self.Underlying().SetIndex), zap.String("handler", r.RequestURI))
			return
		}
		buf := bytes.Buffer{}
		buf.ReadFrom(r.Body)
		defer r.Body.Close()

		vctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := n2nVerifyRequestsWithContext.Run(vctx, func() error {
			if !validateSendRequest(sender, r) {
				return errors.New("failed to validate request")
			}
			return nil
		}); err != nil {
			logging.N2n.Error("message received - failed to validate request",
				zap.String("from", nodeID),
				zap.Int("to", Self.Underlying().SetIndex),
				zap.String("handler", r.RequestURI),
				zap.Error(err))
			return
		}

		entityName := r.Header.Get(HeaderRequestEntityName)
		entityID := r.Header.Get(HeaderRequestEntityID)
		entityMetadata := datastore.GetEntityMetadata(entityName)
		if options != nil && options.MessageFilter != nil {
			if !options.MessageFilter.AcceptMessage(entityName, entityID) {
				return
			}
		}
		ctx := context.Background()
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
		entity, err := getRequestEntity(r, &buf, entityMetadata)
		if err != nil {
			if err == NoDataErr {
				go pullEntityHandler(ctx, sender, r.RequestURI, handler, entityName, entityID)
				sender.AddReceived(1)
				return
			}
			http.Error(w, fmt.Sprintf("Error reading entity: %v", err), 500)
			return
		}
		if entity.GetKey() != entityID {
			logging.N2n.Error("message received - entity id doesn't match with signed id",
				zap.Int("from", sender.SetIndex),
				zap.Int("to", Self.SetIndex),
				zap.String("handler", r.RequestURI),
				zap.String("entity_id", entityID),
				zap.String("entity.id", entity.GetKey()))
			return
		}
		start := time.Now()
		data, err := handler(ctx, entity)
		duration := time.Since(start)
		common.Respond(w, r, data, err)
		if err != nil {
			logging.N2n.Error("message received", zap.Int("from", sender.SetIndex),
				zap.Int("to", Self.Underlying().SetIndex), zap.String("handler", r.RequestURI), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()), zap.Error(err))
		} else {
			logging.N2n.Info("message received", zap.Int("from", sender.SetIndex),
				zap.Int("to", Self.Underlying().SetIndex), zap.String("handler", r.RequestURI), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()))
		}
		sender.AddReceived(1)
	}
}
