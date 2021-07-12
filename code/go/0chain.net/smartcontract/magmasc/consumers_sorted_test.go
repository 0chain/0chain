package magmasc

import (
	"reflect"
	"testing"
)

func Test_consumersSorted_add(t *testing.T) {
	t.Parallel()

	list := consumersSorted{}
	con0 := Consumer{ID: "0"}
	con1 := Consumer{ID: "1"}
	con2 := Consumer{ID: "2"}
	con3 := Consumer{ID: "3"}

	tests := [5]struct {
		name string
		cons *Consumer
		list *consumersSorted
		want []*Consumer
		ret  bool
	}{
		{
			name: "TRUE", // appended
			cons: &con2,
			list: &list,
			want: []*Consumer{&con2},
			ret:  true,
		},
		{
			name: "APPEND", // appended
			cons: &con3,
			list: &list,
			want: []*Consumer{&con2, &con3},
			ret:  true,
		},
		{
			name: "PREPEND", // inserted
			cons: &con0,
			list: &list,
			want: []*Consumer{&con0, &con2, &con3},
			ret:  true,
		},
		{
			name: "INSERT", // inserted
			cons: &con1,
			list: &list,
			want: []*Consumer{&con0, &con1, &con2, &con3},
			ret:  true,
		},
		{
			name: "FALSE", // already have
			cons: &con3,
			list: &list,
			want: []*Consumer{&con0, &con1, &con2, &con3},
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			if got := test.list.add(test.cons); got != test.ret {
				t.Errorf("add() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(test.list.Sorted, test.want) {
				t.Errorf("add() sorted: %#v | want: %#v", test.list.Sorted, test.want)
			}
		})
	}
}

func Test_consumersSorted_get(t *testing.T) {
	t.Parallel()

	list := mockConsumers().Nodes
	cons := list.Sorted[0]

	tests := [2]struct {
		name string
		id   string
		list *consumersSorted
		want *Consumer
		ret  bool
	}{
		{
			name: "TRUE",
			id:   cons.ID,
			list: list,
			want: cons,
			ret:  true,
		},
		{
			name: "FALSE",
			id:   "not_present_id",
			list: list,
			want: nil,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ret := test.list.get(test.id)
			if ret != test.ret {
				t.Errorf("get() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("get() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_consumersSorted_getIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockConsumers().Nodes
	id := list.Sorted[idx].ID

	tests := [2]struct {
		name string
		id   string
		list *consumersSorted
		want int
		ret  bool
	}{
		{
			name: "TRUE",
			id:   id,
			list: list,
			want: idx,
			ret:  true,
		},
		{
			name: "FALSE",
			id:   "not_present_id",
			list: &consumersSorted{},
			want: -1,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ret := test.list.getIndex(test.id)
			if ret != test.ret {
				t.Errorf("getIndex() return: %v | want: %v", got, test.ret)
			}
			if got != test.want {
				t.Errorf("getIndex() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_consumersSorted_remove(t *testing.T) {
	t.Parallel()

	cons := mockConsumer()
	list := &consumersSorted{Sorted: []*Consumer{&cons}}

	tests := [2]struct {
		name string
		id   string
		list *consumersSorted
		want *consumersSorted
		ret  bool
	}{
		{
			name: "TRUE",
			id:   cons.ID,
			list: list,
			want: &consumersSorted{Sorted: make([]*Consumer, 0)},
			ret:  true,
		},
		{
			name: "FALSE",
			id:   "not_present_id",
			list: list,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if ret := test.list.remove(test.id); ret != test.ret {
				t.Errorf("getIndex() return: %v | want: %v", ret, test.ret)
			}
			if test.want != nil && !reflect.DeepEqual(test.list, test.want) {
				t.Errorf("getIndex() sorted: %#v | want: %#v", test.list, test.want)
			}
		})
	}
}

func Test_consumersSorted_removeByIndex(t *testing.T) {
	t.Parallel()

	list := &consumersSorted{
		Sorted: []*Consumer{
			{ID: "0"}, {ID: "1"}, {ID: "2"}, {ID: "3"},
		},
	}

	tests := [4]struct {
		name string
		idx  int
		list *consumersSorted
		want *Consumer
	}{
		{
			name: "OK",
			idx:  2,
			list: list,
			want: &Consumer{ID: "2"},
		},
		{
			name: "OK",
			idx:  2,
			list: list,
			want: &Consumer{ID: "3"},
		},
		{
			name: "OK",
			idx:  0,
			list: list,
			want: &Consumer{ID: "0"},
		},
		{
			name: "OK",
			idx:  0,
			list: list,
			want: &Consumer{ID: "1"},
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			if got := test.list.removeByIndex(test.idx); !reflect.DeepEqual(got, test.want) {
				t.Errorf("removeByIndex() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_consumersSorted_update(t *testing.T) {
	t.Parallel()

	cons := mockConsumer()
	list := &consumersSorted{Sorted: []*Consumer{&cons}}

	tests := [2]struct {
		name string
		cons *Consumer
		list *consumersSorted
		want bool
	}{
		{
			name: "TRUE",
			cons: &cons,
			list: list,
			want: true,
		},
		{
			name: "FALSE",
			cons: &Consumer{ID: "not_present_id"},
			list: list,
			want: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.update(test.cons); got != test.want {
				t.Errorf("update() got: %v | want: %v", got, test.want)
			}
		})
	}
}
