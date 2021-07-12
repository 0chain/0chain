package magmasc

import (
	"reflect"
	"testing"
)

func Test_ProvidersSorted_add(t *testing.T) {
	t.Parallel()

	list := providersSorted{}
	prov0 := Provider{ID: "0"}
	prov1 := Provider{ID: "1"}
	prov2 := Provider{ID: "2"}
	prov3 := Provider{ID: "3"}

	tests := [5]struct {
		name string
		pros *Provider
		list *providersSorted
		want []*Provider
		ret  bool
	}{
		{
			name: "TRUE", // appended
			pros: &prov2,
			list: &list,
			want: []*Provider{&prov2},
			ret:  true,
		},
		{
			name: "APPEND", // appended
			pros: &prov3,
			list: &list,
			want: []*Provider{&prov2, &prov3},
			ret:  true,
		},
		{
			name: "PREPEND", // inserted
			pros: &prov0,
			list: &list,
			want: []*Provider{&prov0, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "INSERT", // inserted
			pros: &prov1,
			list: &list,
			want: []*Provider{&prov0, &prov1, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "FALSE", // already have
			pros: &prov3,
			list: &list,
			want: []*Provider{&prov0, &prov1, &prov2, &prov3},
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			if got := test.list.add(test.pros); got != test.ret {
				t.Errorf("add() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(test.list.Sorted, test.want) {
				t.Errorf("add() sorted: %#v | want: %#v", test.list.Sorted, test.want)
			}
		})
	}
}

func Test_ProvidersSorted_get(t *testing.T) {
	t.Parallel()

	list := mockProviders().Nodes
	pros := list.Sorted[0]

	tests := [2]struct {
		name string
		id   string
		list *providersSorted
		want *Provider
		ret  bool
	}{
		{
			name: "TRUE",
			id:   pros.ID,
			list: list,
			want: pros,
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

func Test_ProvidersSorted_getIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockProviders().Nodes
	id := list.Sorted[idx].ID

	tests := [2]struct {
		name string
		id   string
		list *providersSorted
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
			list: &providersSorted{},
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

func Test_ProvidersSorted_remove(t *testing.T) {
	t.Parallel()

	prov := mockProvider()
	list := &providersSorted{Sorted: []*Provider{&prov}}

	tests := [2]struct {
		name string
		id   string
		list *providersSorted
		want *providersSorted
		ret  bool
	}{
		{
			name: "TRUE",
			id:   prov.ID,
			list: list,
			want: &providersSorted{Sorted: make([]*Provider, 0)},
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

func Test_ProvidersSorted_removeByIndex(t *testing.T) {
	t.Parallel()

	list := &providersSorted{
		Sorted: []*Provider{
			{ID: "0"}, {ID: "1"}, {ID: "2"}, {ID: "3"},
		},
	}

	tests := [4]struct {
		name string
		idx  int
		list *providersSorted
		want *Provider
	}{
		{
			name: "OK",
			idx:  2,
			list: list,
			want: &Provider{ID: "2"},
		},
		{
			name: "OK",
			idx:  2,
			list: list,
			want: &Provider{ID: "3"},
		},
		{
			name: "OK",
			idx:  0,
			list: list,
			want: &Provider{ID: "0"},
		},
		{
			name: "OK",
			idx:  0,
			list: list,
			want: &Provider{ID: "1"},
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

func Test_ProvidersSorted_update(t *testing.T) {
	t.Parallel()

	prov := mockProvider()
	list := &providersSorted{Sorted: []*Provider{&prov}}

	tests := [2]struct {
		name  string
		provs *Provider
		list  *providersSorted
		want  bool
	}{
		{
			name:  "TRUE",
			provs: &prov,
			list:  list,
			want:  true,
		},
		{
			name:  "FALSE",
			provs: &Provider{ID: "not_present_id"},
			list:  list,
			want:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.update(test.provs); got != test.want {
				t.Errorf("update() got: %v | want: %v", got, test.want)
			}
		})
	}
}
