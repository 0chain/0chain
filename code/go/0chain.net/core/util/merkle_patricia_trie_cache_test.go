package util

import (
	"context"
	"testing"
)

func TestMPTCachingProxy_GetNodeValue(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	doStrValInsert(t, wrapped, "01", "1")
	doStrValInsert(t, wrapped, "02", "2")
	doStrValInsert(t, wrapped, "0112", "11")
	doStrValInsert(t, wrapped, "0121", "12")
	doStrValInsert(t, wrapped, "0211", "211")
	doStrValInsert(t, wrapped, "0212", "212")
	doStrValInsert(t, wrapped, "03", "3")
	doStrValInsert(t, wrapped, "0312", "3112")
	doStrValInsert(t, wrapped, "0313", "3113")

	wrapped.GetNodeValue([]byte("01"))

}
