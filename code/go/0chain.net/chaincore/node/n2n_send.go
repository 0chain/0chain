package node

import (
	"bytes"
	"context"
	"errors"
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
	var infos []string
	for _, n := range nodes {
		infos = append(infos, n.GetPseudoName())
	}
	logging.Logger.Debug("send at least", zap.Int("number_to_send", numNodes),
		zap.Int("num_of_nodes", len(nodes)), zap.Strings("nodes", infos))
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
				logging.N2n.Info("send message (done)",
					zap.Int("all_nodes", len(nodes)),
					zap.Int("requested", numNodes),
					zap.Int("active", activeCount),
					zap.Int("sent_to", len(sentTo)),
					zap.Duration("time", time.Since(start)))
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
	return pushTime <= push2pullTime
}

type senderSignInfo struct {
	Ts        common.Timestamp
	TsStr     string
	Hash      string
	Signature string
}

// prepareSenderSign prepare N signature in N seconds
func prepareSenderSign(entity datastore.Entity, num int) ([]*senderSignInfo, error) {
	ts := time.Now()
	ssis := make([]*senderSignInfo, num)
	for i := 0; i < num; i++ {
		t := common.Timestamp(ts.Add(time.Duration(i) * time.Second).Unix())
		hashdata := getHashData(Self.Underlying().GetKey(), t, entity.GetKey())
		hash := encryption.Hash(hashdata)
		signature, err := Self.Sign(hash)
		if err != nil {
			return nil, err
		}

		ssis[i] = &senderSignInfo{
			Ts:        t,
			TsStr:     strconv.FormatInt(int64(t), 10),
			Hash:      hash,
			Signature: signature,
		}
	}
	return ssis, nil
}

/*SendEntityHandler provides a client API to send an entity */
func SendEntityHandler(uri string, options *SendOptions) EntitySendHandler {
	timeout := 500 * time.Millisecond
	if options.Timeout > 0 {
		timeout = options.Timeout
	}
	return func(entity datastore.Entity) SendHandler {
		buf, err := getResponseData(options, entity)
		if err != nil {
			logging.N2n.Error("getResponseData failed", zap.Error(err))
		}

		data := buf.Bytes()

		toPull := options.Pull
		if len(data) > LargeMessageThreshold || toPull {
			toPull = true
			key := p2pKey(uri, entity.GetKey())
			pdce := &pushDataCacheEntry{Options: *options, Data: data, EntityName: entity.GetEntityMetadata().GetName()}
			if err := pushDataCache.Add(key, pdce); err != nil {
				logging.Logger.Error("pull data add to cache failed",
					zap.String("key", key),
					zap.Error(err))
			}
		}

		preparedSignatures, err := prepareSenderSign(entity, 5)
		if err != nil {
			logging.N2n.Panic("failed to prepare sender signature", zap.Error(err))
		}

		setSignHeader := func(r *http.Request) {
			for _, ssi := range preparedSignatures {
				if common.Within(int64(ssi.Ts), int64(time.Second)) {
					r.Header.Set(HeaderRequestTimeStamp, ssi.TsStr)
					r.Header.Set(HeaderRequestHash, ssi.Hash)
					r.Header.Set(HeaderNodeRequestSignature, ssi.Signature)
					return
				}
			}

			// there's no prepared signature within valid time range.
			// generate a new one
			ssis, err := prepareSenderSign(entity, 1)
			if err != nil {
				logging.N2n.Panic("failed to prepare sender signature", zap.Error(err))
			}

			r.Header.Set(HeaderRequestTimeStamp, ssis[0].TsStr)
			r.Header.Set(HeaderRequestHash, ssis[0].Hash)
			r.Header.Set(HeaderNodeRequestSignature, ssis[0].Signature)
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

			if toPull {
				req.Header.Set(HeaderRequestToPull, "true")
			}

			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			SetSendHeaders(req, entity, options)

			setSignHeader(req)
			// Keep the number of messages to a node bounded
			var (
				selfNode *Node
				resp     *http.Response
				ts       = time.Now()
				cctx     context.Context
				cancel   func()
			)

			func() {
				receiver.Grab()
				defer receiver.Release()

				selfNode = Self.Underlying()
				selfNode.SetLastActiveTime(ts)
				selfNode.InduceDelay(receiver)

				cctx, cancel = context.WithTimeout(ctx, timeout)
				req = req.WithContext(cctx)
				resp, err = httpClient.Do(req)
			}()

			defer cancel()

			logging.N2n.Info("sending",
				zap.String("from", selfNode.GetPseudoName()),
				zap.String("to", receiver.GetPseudoName()),
				zap.String("handler", uri),
				zap.Duration("duration", time.Since(ts)),
				zap.String("entity", entity.GetEntityMetadata().GetName()),
				zap.Any("id", entity.GetKey()),
				zap.Any("err", err))
			switch err {
			case nil:
			default:
				ue, ok := err.(*url.Error)
				if ok && ue.Unwrap() != context.Canceled {
					receiver.AddSendErrors(1)
					receiver.AddErrorCount(1)
					logging.N2n.Error("sending", zap.String("from", selfNode.GetPseudoName()), zap.String("to", receiver.GetPseudoName()), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Error(err))
				}
				return false
			}

			receiver.SetStatus(NodeStatusActive)
			receiver.SetLastActiveTime(time.Now())
			receiver.SetErrorCount(receiver.GetSendErrors())

			//TODO may be we don't need to close here, since defer Body.close() is added
			readAndClose(resp.Body)
			if push {
				timer.UpdateSince(ts)
				sizer := receiver.GetSizeMetric(uri)
				sizer.Update(int64(len(data)))
			}
			if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent) {
				logging.N2n.Error("sending", zap.String("from", selfNode.GetPseudoName()), zap.String("to", receiver.GetPseudoName()), zap.String("handler", uri), zap.Duration("duration", time.Since(ts)), zap.String("entity", entity.GetEntityMetadata().GetName()), zap.Any("id", entity.GetKey()), zap.Any("status_code", resp.StatusCode))
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
	selfPseudoName := Self.Underlying().GetPseudoName()
	if !validateChain(sender, r) {
		logging.N2n.Error("message received - invalid chain", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI), zap.String("entity", entityName))
		return false
	}
	if !validateEntityMetadata(sender, r) {
		logging.N2n.Error("message received - invalid entity metadata", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI), zap.String("entity", entityName))
		return false
	}
	if entityID == "" {
		logging.N2n.Error("message received - entity id blank", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI))
		return false
	}
	reqTS := r.Header.Get(HeaderRequestTimeStamp)
	if reqTS == "" {
		logging.N2n.Error("message received - no timestamp for the message", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI), zap.String("entity", entityName), zap.Any("id", entityID))
		return false
	}
	reqTSn, err := strconv.ParseInt(reqTS, 10, 64)
	if err != nil {
		logging.N2n.Error("message received", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI), zap.String("entity", entityName), zap.Any("id", entityID), zap.Error(err))
		return false
	}
	sender.SetStatus(NodeStatusActive)
	sender.SetLastActiveTime(time.Unix(reqTSn, 0))
	Self.Underlying().SetLastActiveTime(time.Now())
	if !common.Within(reqTSn, int64(N2NTimeTolerance*time.Second)) {
		logging.N2n.Error("message received - tolerance", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI),
			zap.String("enitty", entityName), zap.String("id", entityID),
			zap.Int64("ts", reqTSn), zap.Time("tstime", time.Unix(reqTSn, 0)))
		return false
	}

	reqHashdata := getHashData(sender.GetKey(), common.Timestamp(reqTSn), entityID)
	reqHash := r.Header.Get(HeaderRequestHash)
	if reqHash != encryption.Hash(reqHashdata) {
		logging.N2n.Error("message received - request data hash invalid", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI), zap.String("hash", reqHash), zap.String("hashdata", reqHashdata))
		return false
	}
	reqSignature := r.Header.Get(HeaderNodeRequestSignature)
	if ok, _ := sender.Verify(reqSignature, reqHash); !ok {
		logging.N2n.Error("message received - invalid signature", zap.String("from", sender.GetPseudoName()),
			zap.String("to", selfPseudoName), zap.String("handler", r.RequestURI), zap.String("hash", reqHash), zap.String("hashdata", reqHashdata), zap.String("signature", reqSignature))
		return false
	}
	sender.SetStatus(NodeStatusActive)
	sender.SetLastActiveTime(time.Unix(reqTSn, 0))
	return true
}

// Chainer represents an interface that provides chain functions
type Chainer interface {
	// IsBlockSyncing checks if the miner is struggling on syncing
	// previous blocks
	IsBlockSyncing() bool
	// RejectNotarizedBlock returns notarized block if the sharder is processing the block
	// or the block has notarized already
	RejectNotarizedBlock(hash string) bool
}

// StopOnBlockSyncingHandler check if the miner is struggling on syncing blocks,
// which means the CPU usage may be high. In this case, all the requests passed
// in will be ignored.
func StopOnBlockSyncingHandler(c Chainer, handler common.ReqRespHandlerf) common.ReqRespHandlerf {
	return func(writer http.ResponseWriter, request *http.Request) {
		if c.IsBlockSyncing() {
			return
		}

		handler(writer, request)
	}
}

func RejectDuplicateNotarizedBlockHandler(c Chainer, handler common.ReqRespHandlerf) common.ReqRespHandlerf {
	return func(writer http.ResponseWriter, request *http.Request) {
		entityID := request.Header.Get(HeaderRequestEntityID)

		// check if the sharder is processing the block or
		// return if the block is cached
		if c.RejectNotarizedBlock(entityID) {
			return
		}

		handler(writer, request)
	}
}

/*ToN2NReceiveEntityHandler - takes a handler that accepts an entity, processes and responds and converts it
* into something suitable for Node 2 Node communication*/
func ToN2NReceiveEntityHandler(handler datastore.JSONEntityReqResponderF, options *ReceiveOptions) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		contentType := r.Header.Get("Content-type")
		if !strings.HasPrefix(contentType, "application/json") {
			http.Error(w, "Header Content-type=application/json not found", 400)
			return
		}
		nodeID := r.Header.Get(HeaderNodeID)
		sender := GetNode(nodeID)
		if sender == nil {
			logging.N2n.Error("message received - request from unrecognized node",
				zap.String("from", nodeID),
				zap.String("to", Self.Underlying().GetPseudoName()),
				zap.String("handler", r.RequestURI))
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

		buf := bytes.Buffer{}
		if _, err := buf.ReadFrom(r.Body); err != nil {
			logging.N2n.Error("message received - read body failed",
				zap.String("from", nodeID),
				zap.String("to", Self.Underlying().GetPseudoName()),
				zap.String("handler", r.RequestURI),
				zap.Error(err))
		}

		go func() {
			senderValidateFunc := func() error {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				return n2nVerifyRequestsWithContext.Run(ctx, func() error {
					if !validateSendRequest(sender, r) {
						return errors.New("failed to validate request")
					}

					return nil
				})
			}
			// TODO:
			root, _ := context.WithTimeout(common.GetRootContext(), 5*time.Second) //nolint:govet
			ctx := WithSenderValidateFunc(root, senderValidateFunc)
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

			if r.Header.Get(HeaderRequestToPull) == "true" {
				go pullEntityHandler(ctx, sender, r.RequestURI, handler, entityName, entityID)
				sender.AddReceived(1)
				return
			}

			entity, err := getRequestEntity(r, &buf, entityMetadata)
			if err != nil {
				return
			}

			if entity.GetKey() != entityID {
				logging.N2n.Error("message received - entity id doesn't match with signed id",
					zap.String("from", sender.GetPseudoName()),
					zap.String("to", Self.GetPseudoName()),
					zap.String("handler", r.RequestURI),
					zap.String("entity_id", entityID),
					zap.String("entity.id", entity.GetKey()))
				return
			}

			start := time.Now()
			_, err = handler(ctx, entity)
			duration := time.Since(start)
			if err != nil {
				logging.N2n.Error("message received", zap.String("from", sender.GetPseudoName()),
					zap.String("to", Self.Underlying().GetPseudoName()), zap.String("handler", r.RequestURI), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()), zap.Error(err))
			} else {
				logging.N2n.Info("message received", zap.String("from", sender.GetPseudoName()),
					zap.String("to", Self.Underlying().GetPseudoName()), zap.String("handler", r.RequestURI), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()))
			}
			sender.AddReceived(1)

		}()
		common.Respond(w, r, nil, nil)
	}
}

// SenderValidateHandler validates the sender signature
func SenderValidateHandler(handler datastore.JSONEntityReqResponderF) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		if err := ValidateSenderSignature(ctx); err != nil {
			return nil, err
		}

		return handler(ctx, entity)
	}
}
