package node

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

var (
	TimeoutSmallMessage = 1000 * time.Millisecond
	TimeoutLargeMessage = 3000 * time.Millisecond
)

var compDecomp common.CompDe

func init() {
	//compDecomp = common.NewSnappyCompDe()
	compDecomp = common.NewZStdCompDe()
}

//SetTimeoutSmallMessage - set the timeout for small message
func SetTimeoutSmallMessage(ts time.Duration) {
	TimeoutSmallMessage = ts
}

//SetTimeoutLargeMessage - set the timeout for large message
func SetTimeoutLargeMessage(ts time.Duration) {
	TimeoutLargeMessage = ts
}

/*SetupN2NHandlers - Setup all the node 2 node communiations*/
func SetupN2NHandlers() {
	http.HandleFunc("/v1/_n2n/entity/post", ToN2NReceiveEntityHandler(datastore.PrintEntityHandler))
}

var (
	HeaderRequestTimeStamp      = "X-Request-Timestamp"
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

//N2NTimeTolerance - only a message signed within this time is considered valid
const N2NTimeTolerance = 4 // in seconds

const (
	CODEC_JSON    = 0
	CODEC_MSGPACK = 1
)

const (
	CodecJSON    = "JSON"
	CodecMsgpack = "Msgpack"
)

/*SendOptions - options to tune how the messages are sent within the network */
type SendOptions struct {
	Timeout            time.Duration
	MaxRelayLength     int64
	CurrentRelayLength int64
	Compress           bool
	InitialNodeID      string
	CODEC              int
}

var transport *http.Transport
var httpClient *http.Client

func init() {
	transport = &http.Transport{MaxIdleConnsPerHost: 5}
	httpClient = &http.Client{Transport: transport}
}

/*SENDER - key used to get the connection object from the context */
const SENDER common.ContextKey = "node.sender"

/*WithNode takes a context and adds a connection value to it */
func WithNode(ctx context.Context, node *Node) context.Context {
	return context.WithValue(ctx, SENDER, node)
}

/*GetSender returns a connection stored in the context which got created via WithConnection */
func GetSender(ctx context.Context) *Node {
	return ctx.Value(SENDER).(*Node)
}

/*SetHeaders - set common request headers */
func SetHeaders(req *http.Request) {
	req.Header.Set(HeaderRequestChainID, config.GetServerChainID())
	req.Header.Set(HeaderNodeID, Self.GetKey())
}

func getHashData(clientID datastore.Key, ts common.Timestamp, key datastore.Key) string {
	return clientID + ":" + common.TimeToString(ts) + ":" + key
}

func getRequestEntity(r *http.Request, entityMetadata datastore.EntityMetadata) (datastore.Entity, error) {
	defer r.Body.Close()
	var buffer io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "snappy" {
		cbuffer := new(bytes.Buffer)
		cbuffer.ReadFrom(r.Body)
		cbytes, err := compDecomp.Decompress(cbuffer.Bytes())
		if err != nil {
			N2n.Error("snappy decoding", zap.Any("error", err))
			return nil, err
		}
		buffer = bytes.NewReader(cbytes)
	}
	return getEntity(r.Header.Get(HeaderRequestCODEC), buffer, entityMetadata)
}

func getResponseEntity(r *http.Response, entityMetadata datastore.EntityMetadata) (datastore.Entity, error) {
	defer r.Body.Close()
	var buffer io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "snappy" {
		cbuffer := new(bytes.Buffer)
		cbuffer.ReadFrom(r.Body)
		cbytes, err := compDecomp.Decompress(cbuffer.Bytes())
		if err != nil {
			N2n.Error("snappy decoding", zap.Any("error", err))
			return nil, err
		}
		buffer = bytes.NewReader(cbytes)
	}
	return getEntity(r.Header.Get(HeaderRequestCODEC), buffer, entityMetadata)
}

func getEntity(codec string, reader io.Reader, entityMetadata datastore.EntityMetadata) (datastore.Entity, error) {
	entity := entityMetadata.Instance()
	switch codec {
	case CodecMsgpack:
		if err := datastore.FromMsgpack(reader, entity.(datastore.Entity)); err != nil {
			N2n.Error("msgpack decoding", zap.Any("error", err))
			return nil, err
		}
		return entity, nil
	case CodecJSON:
		if err := datastore.FromJSON(reader, entity.(datastore.Entity)); err != nil {
			N2n.Error("json decoding", zap.Any("error", err))
			return nil, err
		}
		return entity, nil
	}
	Logger.Error("uknown_encoding", zap.String("encoding", codec))
	return nil, common.NewError("unkown_encoding", "unknown encoding")
}

func getResponseData(options *SendOptions, entity datastore.Entity) *bytes.Buffer {
	var buffer *bytes.Buffer
	if options.CODEC == datastore.CodecJSON {
		buffer = datastore.ToJSON(entity)
	} else {
		buffer = datastore.ToMsgpack(entity)
	}
	if options.Compress {
		cbytes := compDecomp.Compress(buffer.Bytes())
		buffer = bytes.NewBuffer(cbytes)
	}
	return buffer
}
