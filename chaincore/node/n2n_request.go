package node

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

const (
	FetchStrategyRandom  = 0
	FetchStrategyNearest = 1
)

// FetchStrategy - when fetching an entity, the strategy to use to select the peer nodes
// var FetchStrategy = FetchStrategyNearest
var FetchStrategy = FetchStrategyRandom

// GetFetchStrategy - indicate which fetch strategy to use
func GetFetchStrategy() int {
	if Self.Underlying().Type == NodeTypeSharder {
		return FetchStrategyRandom
	} else {
		return FetchStrategy
	}
}

// RequestEntity - request an entity from nodes in the pool, returns when any node has response
func (np *Pool) RequestEntity(ctx context.Context, requestor EntityRequestor, params *url.Values, handler datastore.JSONEntityReqResponderF) *Node {
	rhandler := requestor(params, handler)
	var nds []*Node
	if GetFetchStrategy() == FetchStrategyRandom {
		nds = np.ShuffleNodes(true)
	} else {
		nds = np.GetNodesByLargeMessageTime()
	}

	var (
		total  = len(nds)
		minNum = 4
		reqNum int
	)

	if total < minNum {
		reqNum = total
	} else {
		reqNum = int(math.Ceil(float64(total) / 10.0))
		if reqNum < minNum {
			reqNum = minNum
		}
	}

	return sendRequestConcurrent(ctx, nds[:reqNum], rhandler)

	//reqNum := (1 / 10) * len(nds)
	//batchSize := 4
	//batchNum := total / batchSize
	//if total%batchSize > 0 {
	//	batchNum++
	//}
	//
	//for i := 0; i < batchNum; i++ {
	//	start := i * batchSize
	//	end := (i + 1) * batchSize
	//	if end > total {
	//		end = total
	//	}
	//
	//	select {
	//	case <-ctx.Done():
	//		logging.Logger.Error("request entity - context done", zap.Error(ctx.Err()))
	//		return nil
	//	default:
	//		n := sendRequestConcurrent(ctx, nds[start:end], rhandler)
	//		if n != nil {
	//			return n
	//		}
	//	}
	//}

	//return nil
}

func sendRequestConcurrent(ctx context.Context, nds []*Node, handler SendHandler) *Node {
	wg := &sync.WaitGroup{}
	nodeC := make(chan *Node, len(nds))
	for _, nd := range nds {
		if nd.GetStatus() == NodeStatusInactive {
			continue
		}
		if Self.IsEqual(nd) {
			continue
		}

		wg.Add(1)
		go func(n *Node) {
			if handler(ctx, n) {
				select {
				case nodeC <- n:
				default:
				}
			}
			wg.Done()
		}(nd)
	}

	wg.Wait()
	close(nodeC)
	n, ok := <-nodeC
	if ok {
		return n
	}

	return nil
}

// RequestEntityFromNodes - request an entity from nodes, returns when any node has response
func RequestEntityFromNodes(ctx context.Context, nds []*Node, requestor EntityRequestor, params *url.Values, handler datastore.JSONEntityReqResponderF) *Node {
	return sendRequestConcurrent(ctx, nds, requestor(params, handler))
}

// RequestEntityFromAll - requests an entity from all the nodes
func (np *Pool) RequestEntityFromAll(ctx context.Context,
	requestor EntityRequestor, params *url.Values,
	handler datastore.JSONEntityReqResponderF) {
	wg := &sync.WaitGroup{}
	rhandler := requestor(params, handler)
	var nodes []*Node
	if GetFetchStrategy() == FetchStrategyRandom {
		nodes = np.ShuffleNodes(true)
	} else {
		nodes = np.GetNodesByLargeMessageTime()
	}
	for _, nd := range nodes {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if nd.GetStatus() == NodeStatusInactive {
			continue
		}
		if Self.IsEqual(nd) {
			continue
		}
		wg.Add(1)
		go func(n *Node) {
			rhandler(ctx, n)
			wg.Done()
		}(nd)
	}
	wg.Wait()
}

// RequestEntityFromNode - request an entity from a node
func (n *Node) RequestEntityFromNode(ctx context.Context, requestor EntityRequestor, params *url.Values, handler datastore.JSONEntityReqResponderF) bool {
	rhandler := requestor(params, handler)
	select {
	case <-ctx.Done():
		logging.Logger.Error("RequestEntityFromNode failed", zap.Error(ctx.Err()))
		return false
	default:
		return rhandler(ctx, n)
	}
}

/*SetRequestHeaders - sets the send request headers*/
func SetRequestHeaders(req *http.Request, options *SendOptions, entityMetadata datastore.EntityMetadata) bool {
	SetHeaders(req)
	if options.InitialNodeID != "" {
		req.Header.Set(HeaderInitialNodeID, options.InitialNodeID)
	}
	if entityMetadata != nil {
		req.Header.Set(HeaderRequestEntityName, entityMetadata.GetName())
	}

	if options.CODEC == 0 {
		req.Header.Set(HeaderRequestCODEC, CodecJSON)
	} else {
		req.Header.Set(HeaderRequestCODEC, CodecMsgpack)
	}
	return true
}

// RequestEntityHandler - a handler that requests an entity and uses it
func RequestEntityHandler(uri string, options *SendOptions, entityMetadata datastore.EntityMetadata) EntityRequestor {
	return func(params *url.Values, handler datastore.JSONEntityReqResponderF) SendHandler {
		return func(ctx context.Context, provider *Node) bool {
			entityMeta := entityMetadata
			timer := provider.GetTimer(uri)
			timeout := 500 * time.Millisecond
			if options.Timeout > 0 {
				timeout = options.Timeout
			}
			u := provider.GetN2NURLBase() + uri
			var data io.Reader
			if params != nil {
				data = strings.NewReader(params.Encode())
			}
			req, err := http.NewRequest("POST", u, data)
			if err != nil {
				return false
			}
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			if options.Compress {
				req.Header.Set("Content-Encoding", compDecomp.Encoding())
			}

			var (
				ts       time.Time
				selfNode = Self.Underlying()
				resp     *http.Response
				cancel   func()
				eName    string
			)

			if entityMeta != nil {
				eName = entityMeta.GetName()
			}

			SetRequestHeaders(req, options, entityMeta)
			// Keep the number of messages to a node bounded

			var (
				tm       *time.Timer
				closeTmC = make(chan struct{})
			)
			func() {
				provider.Grab()
				defer provider.Release()
				ts = time.Now()

				selfNode.SetLastActiveTime(ts)
				selfNode.InduceDelay(provider)

				var cctx context.Context
				tm = time.NewTimer(timeout)
				cctx, cancel = context.WithCancel(ctx)
				go func() {
					select {
					case <-tm.C:
						cancel()
					case <-closeTmC:
					}
				}()
				req = req.WithContext(cctx)
				resp, err = httpClient.Do(req)
			}()
			defer cancel()

			duration := time.Since(ts)
			var buf bytes.Buffer
			switch err {
			case nil:
				if tm != nil {
					tm.Stop()
					close(closeTmC)
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusNotModified {
					provider.SetStatus(NodeStatusActive)
					provider.SetLastActiveTime(time.Now())
					provider.SetErrorCount(provider.GetSendErrors())
					logging.N2n.Debug("requesting - not modified",
						zap.String("from", selfNode.GetPseudoName()),
						zap.String("to", provider.GetPseudoName()),
						zap.Duration("duration", duration),
						zap.String("handler", uri),
						zap.String("entity", eName),
						zap.Any("params", params))
					return true
				}

				// reset context timeout so that the
				// following data reading would not be canceled due to timeout
				_, err := buf.ReadFrom(resp.Body)
				if err != nil {
					logging.N2n.Error("requesting - read response failed",
						zap.String("from", selfNode.GetPseudoName()),
						zap.String("to", provider.GetPseudoName()),
						zap.Duration("duration", duration),
						zap.String("handler", uri),
						zap.String("entity", eName),
						zap.Any("params", params), zap.Error(err))
					return false
				}
			default:
				ue, ok := err.(*url.Error)
				if ok && ue.Unwrap() != context.Canceled {
					// requests could be canceled when the miner has received a response
					// from any of the remotes.
					provider.AddSendErrors(1)
					provider.AddErrorCount(1)
					logging.N2n.Error("requesting", zap.String("from", selfNode.GetPseudoName()),
						zap.String("to", provider.GetPseudoName()), zap.Duration("duration", duration), zap.String("handler", uri), zap.String("entity", eName), zap.Any("params", params), zap.Error(err))
				}
				return false
			}

			// As long as the node is reachable, it is active.
			provider.SetStatus(NodeStatusActive)
			provider.SetLastActiveTime(time.Now())
			provider.SetErrorCount(provider.GetSendErrors())

			if resp.StatusCode != http.StatusOK {
				data := buf.String()
				logging.N2n.Error("requesting",
					zap.String("from", selfNode.GetPseudoName()),
					zap.String("to", provider.GetPseudoName()),
					zap.Duration("duration", duration),
					zap.String("handler", uri),
					zap.String("entity", eName),
					zap.Any("params", params),
					zap.Int("status_code", resp.StatusCode),
					zap.String("response", data))
				return false
			}
			if entityMeta == nil {
				eName = resp.Header.Get(HeaderRequestEntityName)
				if eName == "" {
					logging.N2n.Error("requesting - no entity name in header", zap.String("from", selfNode.GetPseudoName()), zap.String("to", provider.GetPseudoName()), zap.Duration("duration", duration), zap.String("handler", uri))
				}
				logging.N2n.Debug("requesting entityMetadata nil, get from response header",
					zap.String("from", selfNode.GetPseudoName()),
					zap.String("to", provider.GetPseudoName()),
					zap.Duration("duration", duration),
					zap.String("handler", uri),
					zap.Any("params", params),
					zap.String("entity", eName))
				entityMeta = datastore.GetEntityMetadata(eName)
				if entityMeta == nil {
					data := buf.String()
					logging.N2n.Error("requesting - unknown entity",
						zap.String("from", selfNode.GetPseudoName()),
						zap.String("to", provider.GetPseudoName()),
						zap.Duration("duration", duration),
						zap.String("handler", uri),
						zap.String("entity", eName),
						zap.String("response", data))
					return false
				}
			}

			size, entity, err := getResponseEntity(resp, &buf, entityMeta)
			if err != nil {
				logging.N2n.Error("requesting", zap.String("from", selfNode.GetPseudoName()), zap.String("to", provider.GetPseudoName()), zap.Duration("duration", duration), zap.String("handler", uri), zap.String("entity", eName), zap.Any("params", params), zap.Error(err))
				return false
			}
			duration = time.Since(ts)
			timer.UpdateSince(ts)
			sizer := provider.GetSizeMetric(uri)
			sizer.Update(int64(size))
			logging.N2n.Info("requesting",
				zap.String("from", selfNode.GetPseudoName()),
				zap.String("to", provider.GetPseudoName()),
				zap.Duration("duration", duration),
				zap.String("handler", uri),
				zap.String("entity", eName),
				zap.String("id", entity.GetKey()),
				zap.Any("params", params),
				zap.String("codec", resp.Header.Get(HeaderRequestCODEC)))
			_, err = handler(ctx, entity)
			if err != nil {
				logging.N2n.Error("requesting", zap.String("from", selfNode.GetPseudoName()), zap.String("to", provider.GetPseudoName()), zap.Duration("duration", time.Since(ts)), zap.String("handler", uri), zap.String("entity", entityMeta.GetName()), zap.Any("params", params), zap.Error(err))
				return false
			}
			return true
		}
	}
}

func validateRequest(sender *Node, r *http.Request) bool {
	if !validateChain(sender, r) {
		return false
	}
	if !validateEntityMetadata(sender, r) {
		return false
	}
	return true
}

/*ToN2NSendEntityHandler - takes a handler that accepts an entity, processes and responds and converts it
* into something suitable for Node 2 Node communication*/
func ToN2NSendEntityHandler(handler common.JSONResponderF) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.Header.Get(HeaderNodeID)
		sender := GetNode(nodeID)
		if sender == nil {
			logging.N2n.Error("message received - request from unrecognized node", zap.String("from", nodeID),
				zap.String("to", Self.Underlying().GetPseudoName()), zap.String("handler", r.RequestURI))
			return
		}
		if !validateRequest(sender, r) {
			return
		}
		sender.AddReceived(1)
		ctx := context.TODO()
		ts := time.Now()
		data, err := handler(ctx, r)
		if err != nil {
			common.Respond(w, r, nil, err)
			logging.N2n.Error("message received", zap.String("from", sender.GetPseudoName()),
				zap.String("to", Self.Underlying().GetPseudoName()), zap.String("handler", r.RequestURI), zap.Error(err))
			return
		}
		options := &SendOptions{Compress: true}
		var buffer *bytes.Buffer
		uri := r.URL.Path
		switch v := data.(type) {
		case datastore.Entity:
			entity := v
			codec := r.Header.Get(HeaderRequestCODEC)
			switch codec {
			case "JSON":
				options.CODEC = CODEC_JSON
			case "Msgpack":
				options.CODEC = CODEC_MSGPACK
			}
			w.Header().Set(HeaderRequestCODEC, codec)
			buffer, err = getResponseData(options, entity)
			if err != nil {
				logging.N2n.Error("getResponseData failed", zap.Error(err))
				return
			}
		case *pushDataCacheEntry:
			options.CODEC = v.Options.CODEC
			if options.CODEC == 0 {
				w.Header().Set(HeaderRequestCODEC, CodecJSON)
			} else {
				w.Header().Set(HeaderRequestCODEC, CodecMsgpack)
			}
			w.Header().Set(HeaderRequestEntityName, v.EntityName)
			buffer = bytes.NewBuffer(v.Data)
			uri = r.FormValue("_puri")
		}
		if options.Compress {
			w.Header().Set("Content-Encoding", compDecomp.Encoding())
		}
		w.Header().Set("Content-Type", "application/json")
		sData := buffer.Bytes()
		if _, err := w.Write(sData); err != nil {
			logging.N2n.Error("message received - http write failed",
				zap.String("to", Self.Underlying().GetPseudoName()),
				zap.String("handler", r.RequestURI),
				zap.Error(err))
		}

		if isPullRequest(r) {
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			updatePullStats(sender, uri, len(sData), ts)
		}
		logging.N2n.Info("message received", zap.String("from", sender.GetPseudoName()),
			zap.String("to", Self.Underlying().GetPseudoName()),
			zap.String("handler", r.RequestURI),
			zap.Duration("duration", time.Since(ts)),
			zap.Int("codec", options.CODEC))
	}
}

func ToS2MSendEntityHandler(handler common.JSONResponderF) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.TODO()
		ts := time.Now()
		data, err := handler(ctx, r)
		if err != nil {
			common.Respond(w, r, nil, err)
			logging.N2n.Error("message received",
				zap.String("to", Self.Underlying().GetPseudoName()),
				zap.String("handler", r.RequestURI), zap.Error(err))
			return
		}
		options := &SendOptions{Compress: true}
		var buffer *bytes.Buffer
		switch v := data.(type) {
		case datastore.Entity:
			entity := v
			codec := r.Header.Get(HeaderRequestCODEC)
			switch codec {
			case "JSON":
				options.CODEC = CODEC_JSON
			case "Msgpack":
				options.CODEC = CODEC_MSGPACK
			}
			w.Header().Set(HeaderRequestCODEC, codec)
			buffer, err = getResponseData(options, entity)
			if err != nil {
				logging.N2n.Error("getResponseData failed", zap.Error(err))
				return
			}
		case *pushDataCacheEntry:
			options.CODEC = v.Options.CODEC
			if options.CODEC == 0 {
				w.Header().Set(HeaderRequestCODEC, CodecJSON)
			} else {
				w.Header().Set(HeaderRequestCODEC, CodecMsgpack)
			}
			w.Header().Set(HeaderRequestEntityName, v.EntityName)
			buffer = bytes.NewBuffer(v.Data)
		}
		if options.Compress {
			w.Header().Set("Content-Encoding", compDecomp.Encoding())
		}
		w.Header().Set("Content-Type", "application/json")
		sData := buffer.Bytes()
		if _, err := w.Write(sData); err != nil {
			logging.N2n.Error("message received - http write failed",
				zap.String("to", Self.Underlying().GetPseudoName()),
				zap.String("handler", r.RequestURI),
				zap.Error(err))
		}

		if isPullRequest(r) {
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		logging.N2n.Info("message received",
			zap.String("to", Self.Underlying().GetPseudoName()),
			zap.String("handler", r.RequestURI),
			zap.Duration("duration", time.Since(ts)),
			zap.Int("codec", options.CODEC))
	}
}
