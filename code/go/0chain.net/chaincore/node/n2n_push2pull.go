package node

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"go.uber.org/zap"
)

var pushDataCache = cache.NewLRUCache(100)

//pushDataCacheEntry - cached push data
type pushDataCacheEntry struct {
	Options    SendOptions
	Data       []byte
	EntityName string
}

var pullURL = "/v1/n2n/entity_pull/get"

func getPushToPullTime(n *Node) float64 {
	var pullRequestTime float64
	sendTime := n.GetSmallMessageSendTime()
	if pullRequestTimer := n.GetTimer(pullURL); pullRequestTimer != nil && pullRequestTimer.Count() >= 50 {
		pullRequestTime = pullRequestTimer.Mean()
	} else {
		pullRequestTime = 2 * sendTime
	}
	return pullRequestTime + sendTime
}

var pullDataCache = cache.NewLRUCache(100)

type nodeRequest struct {
	node      *Node
	requested bool
}

const (
	pullStatePulling = 1
	pullStateFailed  = iota
	pullStateDone    = iota
)

type pullDataCacheEntry struct {
	sentBy []*nodeRequest
	state  int8
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
		logging.N2n.Error("push to pull", zap.String("key", key), zap.Error(err))
		return nil, common.NewErrorf("request_data_not_found", "Requested data is not found, key: %v", key)
	}
	return pcde, nil
}

var pullLock sync.Mutex

/*pullEntityHandler - pull an entity that wasn't pushed as it's large and pulling is cheaper */
func pullEntityHandler(ctx context.Context, nd *Node, uri string, handler datastore.JSONEntityReqResponderF, entityName string, entityID datastore.Key) {
	phandler := func(pctx context.Context, entity datastore.Entity) (interface{}, error) {
		if entity.GetEntityMetadata().GetName() != entityName {
			return entity, nil
		}
		if entity.GetKey() != entityID {
			return entity, nil
		}
		start := time.Now()
		_, err := handler(pctx, entity)
		duration := time.Since(start)
		if err != nil {
			logging.N2n.Error("message pull", zap.Int("from", nd.SetIndex),
				zap.Int("to", Self.Underlying().SetIndex), zap.String("handler", uri), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()), zap.Error(err))
			return nil, err
		}
		//N2n.Debug("message pull", zap.Int("from", nd.SetIndex), zap.Int("to", Self.Underlying().SetIndex), zap.String("handler", uri), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()))
		return entity, nil
	}
	params := &url.Values{}
	params.Add("__push2pull", "true")
	params.Add("_puri", uri)
	params.Add("id", datastore.ToString(entityID))
	rhandler := pullDataRequestor(params, phandler)

	rhandler(ctx, nd)
}

func isPullRequest(r *http.Request) bool {
	return r.FormValue("__push2pull") == "true"
}

func updatePullStats(sender *Node, uri string, length int, ts time.Time) {
	mkey := serveMetricKey(uri)
	timer := sender.GetTimer(mkey)
	timer.UpdateSince(ts)
	sizer := sender.GetSizeMetric(mkey)
	sizer.Update(int64(length))
}
