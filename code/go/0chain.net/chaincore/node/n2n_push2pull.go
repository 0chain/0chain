package node

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	core_endpoint "0chain.net/core/endpoint"
	"0chain.net/core/logging"
	"go.uber.org/zap"
)

var (
	pushDataCache      = cache.NewLRUCache(100)
	pullingEntityCache = newPullingCache(1000, 5)
)

//pushDataCacheEntry - cached push data
type pushDataCacheEntry struct {
	Options    SendOptions
	Data       []byte
	EntityName string
}

func getPushToPullTime(n *Node) float64 {
	var pullRequestTime float64
	sendTime := n.GetSmallMessageSendTime()
	if pullRequestTimer := n.GetTimer(core_endpoint.NodeToNodeGetEntity); pullRequestTimer != nil && pullRequestTimer.Count() >= 50 {
		pullRequestTime = pullRequestTimer.Mean()
	} else {
		pullRequestTime = 2 * sendTime
	}
	return pullRequestTime + sendTime
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
			logging.N2n.Error("message pull", zap.String("from", nd.GetPseudoName()),
				zap.String("to", Self.Underlying().GetPseudoName()), zap.String("handler", uri), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()), zap.Error(err))
			return nil, err
		}
		//N2n.Debug("message pull", zap.String("from", nd.GetPseudoName()), zap.String("to", Self.Underlying().GetPseudoName()), zap.String("handler", uri), zap.Duration("duration", duration), zap.String("entity", entityName), zap.Any("id", entity.GetKey()))
		return entity, nil
	}
	params := &url.Values{}
	params.Add("__push2pull", "true")
	params.Add("_puri", uri)
	params.Add("id", datastore.ToString(entityID))
	rhandler := pullDataRequestor(params, phandler)

	pullKey := fmt.Sprintf("%s:%s", entityName, entityID)
	// entity with the same key id will be cached till the first request is returned
	pullingEntityCache.pullOrCacheRequest(ctx, pullKey, func(ctx context.Context) bool {
		return rhandler(ctx, nd)
	})
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

// pullingCache represents the cache for pulling request.
// the key is the 'entityName:id', and value is a buffered channel
type pullingCache struct {
	cache *cache.LRU
	mutex sync.Mutex
	// chanSize is the channel buffer size
	chanSize int
}

func newPullingCache(cacheSize, chanSize int) *pullingCache {
	return &pullingCache{
		cache:    cache.NewLRUCache(cacheSize),
		chanSize: chanSize,
	}
}

type pullHandlerFunc func(ctx context.Context) bool

// pullOrCacheRequest checks if the entity id is in the cache, add it if not exist, and return false
// to indicate the entity was not in the cache, otherwise reject it and return true.
func (c *pullingCache) pullOrCacheRequest(ctx context.Context, key string, pullHandler pullHandlerFunc) {
	c.mutex.Lock()
	v, err := c.cache.Get(key)
	switch err {
	case cache.ErrKeyNotFound:
		ch := make(chan pullHandlerFunc, c.chanSize)
		ch <- pullHandler
		if err := c.cache.Add(key, ch); err != nil {
			logging.Logger.Warn("cache pull handler func failed", zap.Error(err))
		}

		c.mutex.Unlock()

		go c.runHandler(ctx, key, ch)
		return
	case nil:
		ch, ok := v.(chan pullHandlerFunc)
		if ok {
			select {
			case ch <- pullHandler:
			default:
			}
		}
	default:
		logging.Logger.Error("Unexpected error on pulling entity", zap.Error(err))
	}
	c.mutex.Unlock()
}

func (c *pullingCache) runHandler(ctx context.Context, key string, ch chan pullHandlerFunc) {
	wg := sync.WaitGroup{}
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// have workers to process the pulling requests concurrently
	for i := 0; i < c.chanSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-cctx.Done():
			case f, ok := <-ch:
				if ok {
					if f(cctx) {
						// remove from cache when process successfully
						c.mutex.Lock()
						if ch != nil {
							close(ch)
							ch = nil
						}

						c.cache.Remove(key)
						c.mutex.Unlock()
						cancel()
						return
					}
				}
			}
		}()
	}

	wg.Wait()
}
