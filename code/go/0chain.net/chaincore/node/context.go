package node

import (
	"context"

	"0chain.net/core/common"
)

// SelfNodeKey - a key for the context value
const SelfNodeKey common.ContextKey = "SELF_NODE"

/*GetNodeContext - setup a context with the self node */
func GetNodeContext() context.Context {
	return context.WithValue(context.Background(), SelfNodeKey, Self)
}

/*GetSelfNode - given a context, return the self node associated with it */
func GetSelfNode(ctx context.Context) *SelfNode {
	return ctx.Value(SelfNodeKey).(*SelfNode)
}
