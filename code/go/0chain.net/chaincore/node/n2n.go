package node

import (
	"context"
	"net/url"

	"0chain.net/core/datastore"
)

/*
SendHandler is used to send any message to a given node

f m n  where f is a function that takes a message m and a node n to send the message to.

When this function is partially applied, we get the send handler, i.e., a closure that has the message and can
be repeatedly applied to different nodes
*/
type SendHandler func(ctx context.Context, n *Node) bool

/*
EntitySendHandler is used to send an entity to a given node

Creates the send handler closure by substituting an entity as a message
*/
type EntitySendHandler func(entity datastore.Entity) SendHandler

/*
EntityRequestor is used to request an entity and handle it

f p h n  where p is the parameters to query the entity being requested, h is the handler that processes the response and n is the node.

Creates the send handler closure using p and h that can be repeatedly applied to different nodes till it succeeds
*/
type EntityRequestor func(urlParams *url.Values, handler datastore.JSONEntityReqResponderF) SendHandler

/*N2N interface - provides the API that are required to communicate between the nodes to implement the blockchain protocol
 */
type N2N interface {
	/* SendAll is expected to eventually result in sending a message to all the nodes. It doesn't require that this call has to send it to
	all nodes. In the simplest case of multi-casting, the SendAll might simply send it to all nodes. However, in more sophisticated protocols
	like Gossip or Hypercube, it may send it only to a subset of nodes but result in eventually all (or majority of) nodes getting the message
	*/
	SendAll(handler SendHandler) []*Node

	//Send a message to a specific node
	SendTo(handler SendHandler, to string) (bool, error)

	//Request a message from any node
	RequestEntity(ctx context.Context, requestor EntityRequestor, params map[string]string, handler datastore.JSONEntityReqResponderF) *Node

	//Request a message from all nodes
	RequestEntityFromAll(ctx context.Context, requestor EntityRequestor, params map[string]string, handler datastore.JSONEntityReqResponderF)

	//Request a message from a specific node
	RequestEntityFromNode(ctx context.Context, requestor EntityRequestor, params map[string]string, handler datastore.JSONEntityReqResponderF)
}
