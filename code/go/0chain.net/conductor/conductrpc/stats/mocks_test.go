package stats

func mockBlockRequest() *BlockRequest {
	return &BlockRequest{
		NodeID:   "node-id",
		Hash:     "hash",
		Round:    5,
		Handler:  "handler",
		SenderID: "sender-id",
	}
}
