package util

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestValueNode_GetNodeType(t *testing.T) {
	type fields struct {
		Value             Serializable
		OriginTrackerNode *OriginTrackerNode
	}
	tests := []struct {
		name   string
		fields fields
		want   byte
	}{
		{
			name: "Test_ValueNode_GetNodeType_OK",
			want: NodeTypeValueNode,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			vn := &ValueNode{
				Value:             tt.fields.Value,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if got := vn.GetNodeType(); got != tt.want {
				t.Errorf("GetNodeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueNode_HasValue(t *testing.T) {
	type fields struct {
		Value             Serializable
		OriginTrackerNode *OriginTrackerNode
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Test_ValueNode_HasValue_FALSE",
			fields: fields{
				Value:             &SecureSerializableValue{Buffer: []byte{}},
				OriginTrackerNode: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			vn := &ValueNode{
				Value:             tt.fields.Value,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if got := vn.HasValue(); got != tt.want {
				t.Errorf("HasValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeafNode_Decode(t *testing.T) {
	type fields struct {
		Path              Path
		Value             *ValueNode
		OriginTrackerNode *OriginTrackerNode
	}
	type args struct {
		buf []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_LeafNode_Decode_OK",
			args: args{buf: []byte("ch:")},
		},
		{
			name:    "Test_LeafNode_Decode_ERR",
			args:    args{buf: []byte{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ln := &LeafNode{
				Path:              tt.fields.Path,
				Value:             tt.fields.Value,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if err := ln.Decode(tt.args.buf); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeafNode_GetValue(t *testing.T) {
	type fields struct {
		Path              Path
		Value             *ValueNode
		OriginTrackerNode *OriginTrackerNode
	}
	tests := []struct {
		name   string
		fields fields
		want   Serializable
	}{
		{
			name: "Test_LeafNode_GetValue_OK",
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ln := &LeafNode{
				Path:              tt.fields.Path,
				Value:             tt.fields.Value,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if got := ln.GetValue(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFullNode_Decode(t *testing.T) {
	type fields struct {
		Children          [16][]byte
		Value             *ValueNode
		OriginTrackerNode *OriginTrackerNode
	}
	type args struct {
		buf []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_FullNode_Decode_Invalid_Decoding_ERR",
			args:    args{buf: []byte{}},
			wantErr: true,
		},
		{
			name:    "Test_FullNode_Decode_Hex_Decoding_ERR",
			args:    args{buf: []byte("h:!")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := &FullNode{
				Children:          tt.fields.Children,
				Value:             tt.fields.Value,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if err := fn.Decode(tt.args.buf); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFullNode_index(t *testing.T) {
	type fields struct {
		Children          [16][]byte
		Value             *ValueNode
		OriginTrackerNode *OriginTrackerNode
	}
	type args struct {
		c byte
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      byte
		wantPanic bool
	}{
		{
			name:      "Test_FullNode_index_OK",
			args:      args{c: 58},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("index() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			fn := &FullNode{
				Children:          tt.fields.Children,
				Value:             tt.fields.Value,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if got := fn.index(tt.args.c); got != tt.want {
				t.Errorf("index() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFullNode_GetValue(t *testing.T) {
	type fields struct {
		Children          [16][]byte
		Value             *ValueNode
		OriginTrackerNode *OriginTrackerNode
	}
	tests := []struct {
		name   string
		fields fields
		want   Serializable
	}{
		{
			name: "Test_FullNode_GetValue_OK",
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fn := &FullNode{
				Children:          tt.fields.Children,
				Value:             tt.fields.Value,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if got := fn.GetValue(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtensionNode_Decode(t *testing.T) {
	type fields struct {
		Path              Path
		NodeKey           Key
		OriginTrackerNode *OriginTrackerNode
	}
	type args struct {
		buf []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_ExtensionNode_Decode_OK",
			args:    args{buf: []byte{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			en := &ExtensionNode{
				Path:              tt.fields.Path,
				NodeKey:           tt.fields.NodeKey,
				OriginTrackerNode: tt.fields.OriginTrackerNode,
			}
			if err := en.Decode(tt.args.buf); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetValueNode(t *testing.T) {
	ln := NewLeafNode(nil, 0, NewValueNode())
	fn := NewFullNode(NewValueNode())

	type args struct {
		node Node
	}
	tests := []struct {
		name string
		args args
		want *ValueNode
	}{
		{
			name: "Test_GetValueNode_Nil_Node_OK",
			args: args{node: nil},
			want: nil,
		},
		{
			name: "Test_GetValueNode_Value_Node_OK",
			args: args{node: NewValueNode()},
			want: NewValueNode(),
		},
		{
			name: "Test_GetValueNode_Leaf_Node_OK",
			args: args{node: ln},
			want: ln.Value,
		},
		{
			name: "Test_GetValueNode_Full_Node_OK",
			args: args{node: fn},
			want: fn.Value,
		},
		{
			name: "Test_GetValueNode_Unknown_Node_OK",
			args: args{node: NewExtensionNode(nil, nil)},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetValueNode(tt.args.node); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValueNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateNode(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0})

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name      string
		args      args
		want      Node
		wantErr   bool
		wantPanic bool
	}{
		{
			name:      "Test_CreateNode_Panic",
			args:      args{r: buf},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("index() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got, err := CreateNode(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateNode() got = %v, want %v", got, tt.want)
			}
		})
	}
}
