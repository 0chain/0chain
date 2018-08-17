package node

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

//EntityReceiveHandler - given a key, handles an associated entity
type EntityReceiveHandler func(params map[string]string, handler datastore.JSONEntityReqResponderF) SendHandler

//RequestEntity - request an entity
func (np *Pool) RequestEntity(ctx context.Context, handler SendHandler) *Node {
	nodes := np.shuffleNodes()
	for _, nd := range nodes {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if nd.Status == NodeStatusInactive {
			continue
		}
		if nd == Self.Node {
			continue
		}
		if handler(nd) {
			return nd
		}
	}
	return nil
}

/*SetRequestHeaders - sets the send request headers*/
func SetRequestHeaders(req *http.Request, options *SendOptions, entityMetadata datastore.EntityMetadata) bool {
	SetHeaders(req)
	if options.InitialNodeID != "" {
		req.Header.Set(HeaderInitialNodeID, options.InitialNodeID)
	}
	req.Header.Set(HeaderRequestEntityName, entityMetadata.GetName())

	if options.CODEC == 0 {
		req.Header.Set(HeaderRequestCODEC, CodecJSON)
	} else {
		req.Header.Set(HeaderRequestCODEC, CodecMsgpack)
	}
	return true
}

//RequestEntityHandler - a handler that requests an entity and uses it
func RequestEntityHandler(uri string, options *SendOptions, entityMetadata datastore.EntityMetadata) EntityReceiveHandler {
	return func(params map[string]string, handler datastore.JSONEntityReqResponderF) SendHandler {
		return func(receiver *Node) bool {
			timer := receiver.GetTimer(uri)
			timeout := 500 * time.Millisecond
			if options.Timeout > 0 {
				timeout = options.Timeout
			}
			url := receiver.GetN2NURLBase() + uri
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return false
			}
			q := req.URL.Query()
			for k, v := range params {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
			if options.Compress {
				req.Header.Set("Content-Encoding", "snappy")
			}
			delay := common.InduceDelay()
			N2n.Debug("requesting", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Any("handler", uri), zap.Any("entity", entityMetadata.GetName()), zap.Any("params", params), zap.Any("delay", delay))
			SetRequestHeaders(req, options, entityMetadata)
			ctx, cancel := context.WithCancel(context.TODO())
			req = req.WithContext(ctx)
			time.AfterFunc(timeout, cancel)
			// Keep the number of messages to a node bounded
			receiver.Grab()
			ts := time.Now()
			resp, err := httpClient.Do(req)
			receiver.Release()
			timer.UpdateSince(ts)
			N2n.Info("requesting", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Duration("duration", time.Since(ts)), zap.Any("handler", uri), zap.Any("entity", entityMetadata.GetName()), zap.Any("params", params))

			if err != nil {
				N2n.Error("requesting", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Duration("duration", time.Since(ts)), zap.Any("handler", uri), zap.Any("entity", entityMetadata.GetName()), zap.Any("params", params), zap.Error(err))
				return false
			}
			if resp.StatusCode != http.StatusOK {
				var rbuf bytes.Buffer
				rbuf.ReadFrom(resp.Body)
				resp.Body.Close()
				N2n.Error("requesting", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Duration("duration", time.Since(ts)), zap.Any("handler", uri), zap.Any("entity", entityMetadata.GetName()), zap.Any("params", params), zap.Any("status_code", resp.StatusCode), zap.Any("response", rbuf.String()))
				return false
			}
			receiver.Status = NodeStatusActive
			receiver.LastActiveTime = time.Now()
			entity, err := getResponseEntity(resp, entityMetadata)
			if err != nil {
				N2n.Error("requesting", zap.Any("from", Self.SetIndex), zap.Any("to", receiver.SetIndex), zap.Duration("duration", time.Since(ts)), zap.Any("handler", uri), zap.Any("entity", entityMetadata.GetName()), zap.Any("params", params), zap.Error(err))
				return false
			}
			entity.ComputeProperties()
			ctx = context.TODO()
			if delay > 0 {
				N2n.Debug("response received", zap.Any("from", receiver.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", uri), zap.Any("entity", entityMetadata.GetName()), zap.Any("params", params), zap.Any("delay", delay))
			}
			_, err = handler(ctx, entity)
			if err != nil {
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
* into somethign suitable for Node 2 Node communication*/
func ToN2NSendEntityHandler(handler common.JSONResponderF) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.Header.Get(HeaderNodeID)
		sender := GetNode(nodeID)
		if sender == nil {
			N2n.Error("message received", zap.Any("from", nodeID), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Any("error", "request from unrecognized node"))
			return
		}
		if !validateRequest(sender, r) {
			return
		}
		ctx := context.TODO()
		data, err := handler(ctx, r)
		if err != nil {
			common.Respond(w, nil, err)
			N2n.Error("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI), zap.Error(err))
		} else {
			entity, ok := data.(datastore.Entity)
			if !ok {
				N2n.Error("message received", zap.String("type", fmt.Sprintf("%T", data)))
				return
			}
			options := &SendOptions{Compress: true}
			codec := r.Header.Get(HeaderRequestCODEC)
			switch codec {
			case "JSON":
				options.CODEC = CODEC_JSON
			case "Msgpack":
				options.CODEC = CODEC_MSGPACK
			}
			buffer := getResponseData(options, entity)
			if options.Compress {
				w.Header().Set("Content-Encoding", "snappy")
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set(HeaderRequestCODEC, codec)
			if err != nil {
				if cerr, ok := err.(*common.Error); ok {
					w.Header().Set(common.AppErrorHeader, cerr.Code)
				}
				http.Error(w, err.Error(), 400)
				return
			}
			w.Write(buffer.Bytes())
			N2n.Info("message received", zap.Any("from", sender.SetIndex), zap.Any("to", Self.SetIndex), zap.Any("handler", r.RequestURI))
		}
		sender.Received++
	}
}
