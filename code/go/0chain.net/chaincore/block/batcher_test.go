package block

import (
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"reflect"
	"strconv"
	"testing"
)

func TestRankedTx_HasContention(t1 *testing.T) {
	type fields struct {
		rank        int
		rset        map[datastore.Key]bool
		wset        map[datastore.Key]bool
		Transaction *transaction.Transaction
	}
	type args struct {
		sets []map[datastore.Key]bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "empty sets no contention",
			fields: struct {
				rank        int
				rset        map[datastore.Key]bool
				wset        map[datastore.Key]bool
				Transaction *transaction.Transaction
			}{rank: 0, rset: nil, wset: nil, Transaction: &transaction.Transaction{}},
			args: struct{ sets []map[datastore.Key]bool }{sets: []map[datastore.Key]bool{{"1": true}}},
			want: false,
		},
		{name: "empty only Wsets no contention",
			fields: struct {
				rank        int
				rset        map[datastore.Key]bool
				wset        map[datastore.Key]bool
				Transaction *transaction.Transaction
			}{rank: 0, rset: map[datastore.Key]bool{"1": true}, wset: nil, Transaction: &transaction.Transaction{}},
			args: struct{ sets []map[datastore.Key]bool }{sets: []map[datastore.Key]bool{{"1": true}}},
			want: false,
		},
		{name: "large wset, one value input contention",
			fields: struct {
				rank        int
				rset        map[datastore.Key]bool
				wset        map[datastore.Key]bool
				Transaction *transaction.Transaction
			}{rank: 0,
				rset:        map[datastore.Key]bool{"10": true},
				wset:        map[datastore.Key]bool{"1": true, "2": true, "3": true, "4": true},
				Transaction: &transaction.Transaction{}},
			args: struct{ sets []map[datastore.Key]bool }{sets: []map[datastore.Key]bool{{"1": true}}},
			want: true,
		},
		{name: "large wset, nil value input contention",
			fields: struct {
				rank        int
				rset        map[datastore.Key]bool
				wset        map[datastore.Key]bool
				Transaction *transaction.Transaction
			}{rank: 0,
				rset:        map[datastore.Key]bool{"10": true},
				wset:        map[datastore.Key]bool{"1": true, "2": true, "3": true, "4": true},
				Transaction: &transaction.Transaction{}},
			args: struct{ sets []map[datastore.Key]bool }{nil},
			want: false,
		},
		{name: "large wset, large value input contention",
			fields: struct {
				rank        int
				rset        map[datastore.Key]bool
				wset        map[datastore.Key]bool
				Transaction *transaction.Transaction
			}{rank: 0,
				rset:        map[datastore.Key]bool{"10": true},
				wset:        map[datastore.Key]bool{"1": true, "2": true, "3": true, "4": true},
				Transaction: &transaction.Transaction{}},
			args: struct {
				sets []map[datastore.Key]bool
			}{sets: []map[datastore.Key]bool{{"1": true, "5": true, "6": true}}},
			want: true,
		},
		{name: "large wset, several large value input contention",
			fields: struct {
				rank        int
				rset        map[datastore.Key]bool
				wset        map[datastore.Key]bool
				Transaction *transaction.Transaction
			}{rank: 0,
				rset:        map[datastore.Key]bool{"10": true},
				wset:        map[datastore.Key]bool{"1": true, "2": true, "3": true, "4": true},
				Transaction: &transaction.Transaction{}},
			args: struct {
				sets []map[datastore.Key]bool
			}{sets: []map[datastore.Key]bool{
				{"11": true, "5": true, "6": true},
				{"3": true, "4": true, "12": true},
				{"31": true, "34": true, "35": true},
			},
			},
			want: true,
		},
		{name: "large wset, several large value input no contention",
			fields: struct {
				rank        int
				rset        map[datastore.Key]bool
				wset        map[datastore.Key]bool
				Transaction *transaction.Transaction
			}{rank: 0,
				rset:        map[datastore.Key]bool{"10": true},
				wset:        map[datastore.Key]bool{"1": true, "2": true, "3": true, "4": true},
				Transaction: &transaction.Transaction{}},
			args: struct {
				sets []map[datastore.Key]bool
			}{sets: []map[datastore.Key]bool{
				{"11": true, "5": true, "6": true},
				{"13": true, "14": true, "12": true},
				{"31": true, "34": true, "35": true},
			},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &RankedTx{
				rank:        tt.fields.rank,
				rset:        tt.fields.rset,
				wset:        tt.fields.wset,
				Transaction: tt.fields.Transaction,
			}
			if got := t.HasContention(tt.args.sets...); got != tt.want {
				t1.Errorf("HasContention() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContentionFreeBatcher_Batch(t *testing.T) {
	type fields struct {
		batchSize int
	}
	type args struct {
		b *Block
	}

	txs := make([]*transaction.Transaction, 15)
	for i := 0; i < len(txs); i++ {
		txs[i] = &transaction.Transaction{HashIDField: datastore.HashIDField{Hash: strconv.Itoa(i)}}
	}

	/* Transaction dependencies map, every dependency is a contention on dependent tx
		0---------------->6---------------->10

		1---------------->7---------------\
	                                       \
		2---------------->8----------------->11-------------->12
	                                       /
		3---------------------------------/------------------>13
	                                                        /
		4---------------->9--------------------------------/

		5---------------->14
	*/
	accessMap := map[datastore.Key]*AccessList{
		"0": {
			Reads:  []datastore.Key{"a6"},
			Writes: []datastore.Key{},
		},
		"1": {
			Reads:  []datastore.Key{"a0"},
			Writes: []datastore.Key{"a7"},
		},
		"2": {
			Reads:  []datastore.Key{"a8"},
			Writes: nil,
		},
		"3": {
			Reads:  []datastore.Key{"a2", "a4"},
			Writes: []datastore.Key{"a11", "a13"},
		},
		"4": {
			Reads:  []datastore.Key{"a9"},
			Writes: []datastore.Key{"a9"},
		},
		"5": {
			Reads:  nil,
			Writes: []datastore.Key{"a14"},
		},
		"6": {
			Reads:  []datastore.Key{"a10"},
			Writes: []datastore.Key{"a6"},
		},
		"7": {
			Reads:  nil,
			Writes: []datastore.Key{"a7", "a11"},
		},
		"8": {
			Reads:  []datastore.Key{"a11"},
			Writes: []datastore.Key{"a8"},
		},
		"9": {
			Reads:  nil,
			Writes: []datastore.Key{"a9", "a13"},
		},
		"10": {
			Reads:  nil,
			Writes: []datastore.Key{"a10"},
		},
		"11": {
			Reads:  nil,
			Writes: []datastore.Key{"a11", "a12"},
		},
		"12": {
			Reads:  nil,
			Writes: []datastore.Key{"a12"},
		},
		"13": {
			Reads:  []datastore.Key{"a2"},
			Writes: []datastore.Key{"a13"},
		},
		"14": {
			Reads:  []datastore.Key{"a2"},
			Writes: []datastore.Key{"a14"},
		},
	}

	var batchedByOne [][]*transaction.Transaction
	for _, tx := range txs {
		batchedByOne = append(batchedByOne, []*transaction.Transaction{tx})
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantRet [][]*transaction.Transaction
	}{
		{name: "old block without access list", fields: struct{ batchSize int }{batchSize: 3}, args: struct{ b *Block }{b: &Block{
			UnverifiedBlockBody: UnverifiedBlockBody{
				AccessMap: nil,
				Txns:      txs,
			},
		}}, wantRet: batchedByOne},
		{name: "old block with empty access list", fields: struct{ batchSize int }{batchSize: 3}, args: struct{ b *Block }{b: &Block{
			UnverifiedBlockBody: UnverifiedBlockBody{
				AccessMap: make(map[datastore.Key]*AccessList),
				Txns:      txs,
			},
		}}, wantRet: batchedByOne},
		{name: "block without transactions", fields: struct{ batchSize int }{batchSize: 3}, args: struct{ b *Block }{b: &Block{
			UnverifiedBlockBody: UnverifiedBlockBody{
				AccessMap: make(map[datastore.Key]*AccessList),
				Txns:      make([]*transaction.Transaction, 0),
			},
		}}, wantRet: [][]*transaction.Transaction{}},
		{name: "block with reach transaction list", fields: struct{ batchSize int }{batchSize: 4}, args: struct{ b *Block }{b: &Block{
			UnverifiedBlockBody: UnverifiedBlockBody{
				AccessMap: accessMap,
				Txns:      txs,
			},
		}}, wantRet: [][]*transaction.Transaction{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "block with reach transaction list" {
				c := &ContentionFreeBatcher{
					batchSize: tt.fields.batchSize,
				}

				gotRet := c.Batch(tt.args.b)
				zero := map[string]bool{"0": true, "1": true, "2": true, "3": true, "4": true, "5": true}
				one := map[string]bool{"6": true, "7": true, "8": true, "9": true, "14": true}
				two := map[string]bool{"10": true, "11": true, "13": true}
				three := map[string]bool{"12": true}

				for n, batch := range gotRet {
					for _, tx := range batch {
						switch n {
						case 0, 1:
							delete(zero, tx.GetKey())
						case 2, 3:
							delete(one, tx.GetKey())
						case 4:
							delete(two, tx.GetKey())
						case 5:
							delete(three, tx.GetKey())
						}

					}
				}

				if len(zero) != 0 || len(one) != 0 || len(two) != 0 || len(three) != 0 {
					t.Errorf("Batch() = %v, want %v", gotRet, tt.wantRet)
				}
			} else {
				c := &ContentionFreeBatcher{
					batchSize: tt.fields.batchSize,
				}
				if gotRet := c.Batch(tt.args.b); !reflect.DeepEqual(gotRet, tt.wantRet) {
					t.Errorf("Batch() = %v, want %v", gotRet, tt.wantRet)
				}
			}
		})
	}
}
