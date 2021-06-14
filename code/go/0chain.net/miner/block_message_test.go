package miner

//func TestBlockMessage_Retry(t *testing.T) {
//	type fields struct {
//		Type                    int
//		Sender                  *node.Node
//		Round                   *Round
//		Block                   *block.Block
//		BlockVerificationTicket *block.BlockVerificationTicket
//		Notarization            *Notarization
//		Timestamp               time.Time
//		RetryCount              int8
//		VRFShare                *round.VRFShare
//	}
//	type args struct {
//		bmc chan *BlockMessage
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			bm := &BlockMessage{
//				Type:                    tt.fields.Type,
//				Sender:                  tt.fields.Sender,
//				Round:                   tt.fields.Round,
//				Block:                   tt.fields.Block,
//				BlockVerificationTicket: tt.fields.BlockVerificationTicket,
//				Notarization:            tt.fields.Notarization,
//				Timestamp:               tt.fields.Timestamp,
//				RetryCount:              tt.fields.RetryCount,
//				VRFShare:                tt.fields.VRFShare,
//			}
//		})
//	}
//}
//
//func TestBlockMessage_ShouldRetry(t *testing.T) {
//	type fields struct {
//		Type                    int
//		Sender                  *node.Node
//		Round                   *Round
//		Block                   *block.Block
//		BlockVerificationTicket *block.BlockVerificationTicket
//		Notarization            *Notarization
//		Timestamp               time.Time
//		RetryCount              int8
//		VRFShare                *round.VRFShare
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			bm := &BlockMessage{
//				Type:                    tt.fields.Type,
//				Sender:                  tt.fields.Sender,
//				Round:                   tt.fields.Round,
//				Block:                   tt.fields.Block,
//				BlockVerificationTicket: tt.fields.BlockVerificationTicket,
//				Notarization:            tt.fields.Notarization,
//				Timestamp:               tt.fields.Timestamp,
//				RetryCount:              tt.fields.RetryCount,
//				VRFShare:                tt.fields.VRFShare,
//			}
//			if got := bm.ShouldRetry(); got != tt.want {
//				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestGetMessageLookup(t *testing.T) {
//	type args struct {
//		msgType int
//	}
//	tests := []struct {
//		name string
//		args args
//		want *common.Lookup
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := GetMessageLookup(tt.args.msgType); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetMessageLookup() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestNewBlockMessage(t *testing.T) {
//	type args struct {
//		messageType int
//		sender      *node.Node
//		round       *Round
//		block       *block.Block
//	}
//	tests := []struct {
//		name string
//		args args
//		want *BlockMessage
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := NewBlockMessage(tt.args.messageType, tt.args.sender, tt.args.round, tt.args.block); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewBlockMessage() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
