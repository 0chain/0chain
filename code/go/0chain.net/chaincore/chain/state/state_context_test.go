package state

import (
	"fmt"
	"testing"

	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.InitLogging("development", "")
}

func TestGetItemsByIDs(t *testing.T) {
	type testItem struct {
		ID    string
		Value string
	}

	items := make(map[string]*testItem, 10)
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("t%d", i)
		items[id] = &testItem{
			ID:    id,
			Value: fmt.Sprintf("v%d", i),
		}
	}

	type args struct {
		ids      []string
		getItem  GetItemFunc[*testItem]
		balances CommonStateContextI
	}
	tests := []struct {
		name string
		args args
		want []*testItem
		err  error
	}{
		{
			name: "get one item",
			args: args{
				ids: []string{"t1"},
				getItem: func(id string, _ CommonStateContextI) (*testItem, error) {
					return items["t1"], nil
				},
			},
			want: []*testItem{
				{
					ID:    "t1",
					Value: "v1",
				},
			},
		},
		{
			name: "get 5 item",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ CommonStateContextI) (*testItem, error) {
					return items[id], nil
				},
			},
			want: []*testItem{
				{
					ID:    "t1",
					Value: "v1",
				},
				{
					ID:    "t2",
					Value: "v2",
				},
				{
					ID:    "t3",
					Value: "v3",
				},
				{
					ID:    "t4",
					Value: "v4",
				},
				{
					ID:    "t5",
					Value: "v5",
				},
			},
		},
		{
			name: "get node not found error",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ CommonStateContextI) (*testItem, error) {
					if id == "t2" {
						return nil, util.ErrNodeNotFound
					}
					return items[id], nil
				},
			},
			err: util.ErrNodeNotFound,
		},
		{
			name: "get node not found and value not present errors",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ CommonStateContextI) (*testItem, error) {
					if id == "t2" {
						return nil, util.ErrNodeNotFound
					}

					if id == "t1" {
						return nil, util.ErrValueNotPresent
					}

					return items[id], nil
				},
			},
			err: util.ErrNodeNotFound,
		},
		{
			name: "get value not present error",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ CommonStateContextI) (*testItem, error) {
					if id == "t1" {
						return nil, util.ErrValueNotPresent
					}

					return items[id], nil
				},
			},
			err: fmt.Errorf("could not get item %q: %v", "t1", util.ErrValueNotPresent),
		},
		{
			name: "get multiple value not present errors",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ CommonStateContextI) (*testItem, error) {
					if id == "t1" {
						return nil, util.ErrValueNotPresent
					}

					if id == "t5" {
						return nil, util.ErrValueNotPresent
					}

					return items[id], nil
				},
			},
			err: fmt.Errorf("could not get item %q: %v", "t1", util.ErrValueNotPresent),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetItemsByIDs[*testItem](tc.args.ids, tc.args.getItem, tc.args.balances)
			require.Equal(t, tc.err, err)
			if err != nil {
				return
			}

			require.Equal(t, tc.want, got)
		})
	}
}
