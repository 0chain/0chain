package stats

func mockBlockRequest() *BlockRequest {
	return &BlockRequest{
		NodeID:   "node-id",
		Hash:     "hash",
		Round:    5,
		SenderID: "sender-id",
	}
}
