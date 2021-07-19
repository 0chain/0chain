package magmasc

import (
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"
)

func Test_ProvidersSorted_add(t *testing.T) {
	t.Parallel()

	list := providersSorted{}
	prov0 := bmp.Provider{ExtID: "0"}
	prov1 := bmp.Provider{ExtID: "1"}
	prov2 := bmp.Provider{ExtID: "2"}
	prov3 := bmp.Provider{ExtID: "3"}

	tests := [5]struct {
		name string
		pros *bmp.Provider
		list *providersSorted
		want []*bmp.Provider
		ret  bool
	}{
		{
			name: "TRUE", // appended
			pros: &prov2,
			list: &list,
			want: []*bmp.Provider{&prov2},
			ret:  true,
		},
		{
			name: "APPEND", // appended
			pros: &prov3,
			list: &list,
			want: []*bmp.Provider{&prov2, &prov3},
			ret:  true,
		},
		{
			name: "PREPEND", // inserted
			pros: &prov0,
			list: &list,
			want: []*bmp.Provider{&prov0, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "INSERT", // inserted
			pros: &prov1,
			list: &list,
			want: []*bmp.Provider{&prov0, &prov1, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "FALSE", // already have
			pros: &prov3,
			list: &list,
			want: []*bmp.Provider{&prov0, &prov1, &prov2, &prov3},
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
		want *bmp.Provider
		ret  bool
	}{
		{
			name: "TRUE",
			id:   pros.ExtID,
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
	id := list.Sorted[idx].ExtID

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
	list := &providersSorted{Sorted: []*bmp.Provider{prov}}

	tests := [2]struct {
		name string
		id   string
		list *providersSorted
		want *providersSorted
		ret  bool
	}{
		{
			name: "TRUE",
			id:   prov.ExtID,
			list: list,
			want: &providersSorted{Sorted: make([]*bmp.Provider, 0)},
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
		Sorted: []*bmp.Provider{
			{ExtID: "0"}, {ExtID: "1"}, {ExtID: "2"}, {ExtID: "3"},
		},
	}

	tests := [4]struct {
		name string
		idx  int
		list *providersSorted
		want *bmp.Provider
	}{
		{
			name: "OK",
			idx:  2,
			list: list,
			want: &bmp.Provider{ExtID: "2"},
		},
		{
			name: "OK",
			idx:  2,
			list: list,
			want: &bmp.Provider{ExtID: "3"},
		},
		{
			name: "OK",
			idx:  0,
			list: list,
			want: &bmp.Provider{ExtID: "0"},
		},
		{
			name: "OK",
			idx:  0,
			list: list,
			want: &bmp.Provider{ExtID: "1"},
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
	list := &providersSorted{Sorted: []*bmp.Provider{prov}}

	tests := [2]struct {
		name string
		prov *bmp.Provider
		list *providersSorted
		want bool
	}{
		{
			name: "TRUE",
			prov: prov,
			list: list,
			want: true,
		},
		{
			name: "FALSE",
			prov: &bmp.Provider{ExtID: "not_present_id"},
			list: list,
			want: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.update(test.prov); got != test.want {
				t.Errorf("update() got: %v | want: %v", got, test.want)
			}
		})
	}
}
