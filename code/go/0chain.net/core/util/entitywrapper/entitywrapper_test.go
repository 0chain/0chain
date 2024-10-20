package entitywrapper

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type Foo struct {
	Wrapper
}

// func (f *Foo) MarshalMsg(b []byte) ([]byte, error) {
// 	return f.Wrapper.MarshalMsg(b)
// }

func (f *Foo) UnmarshalMsg(b []byte) ([]byte, error) {
	return f.UnmarshalMsgType(b, f.TypeName())
}

func (f *Foo) UnmarshalJSON(b []byte) error {
	return f.UnmarshalJSONType(b, f.TypeName())
}

func (f *Foo) TypeName() string {
	return "Foo"
}

func TestEntityWrapper(t *testing.T) {
	RegisterWrapper(&Foo{}, map[string]EntityI{
		DefaultOriginVersion: &foo{},
		"v2":                 &fooV2{},
		"v3":                 &fooV3{},
	})

	fv1 := foo{
		ID: "foo_id",
	}

	fv1Data, err := fv1.MarshalMsg(nil)
	require.NoError(t, err)

	fooWp := &Foo{}
	_, err = fooWp.UnmarshalMsg(fv1Data)
	require.NoError(t, err)

	vfv1, ok := fooWp.Entity().(*foo)
	require.True(t, ok)
	require.Equal(t, "foo_id", vfv1.ID)

	// migrate foo to fooV2 and save entity
	fv2 := fooV2{
		Version: "v2",
		ID:      vfv1.ID,
		Name:    "foo_name",
	}

	fooWp.SetEntity(&fv2)

	v2Data, err := fooWp.MarshalMsg(nil)
	require.NoError(t, err)

	// // load and unmarshal data from v2Data
	fooWp2 := &Foo{}
	_, err = fooWp2.UnmarshalMsg(v2Data)
	require.NoError(t, err)

	vfv2, ok := fooWp2.Entity().(*fooV2)
	require.True(t, ok)
	require.Equal(t, "v2", vfv2.Version)
	require.Equal(t, "foo_id", vfv2.ID)
	require.Equal(t, "foo_name", vfv2.Name)

	vfooV3 := &fooV3{
		Version: "v3",
		ID:      "foo_id",
		Age:     100,
	}

	fooWp2.SetEntity(vfooV3)
	v3Data, err := fooWp2.MarshalMsg(nil)
	require.NoError(t, err)

	fooWp3 := Foo{}
	require.NoError(t, err)

	_, err = fooWp3.UnmarshalMsg(v3Data)
	require.NoError(t, err)

	vfv3, ok := fooWp3.Entity().(*fooV3)
	require.True(t, ok)
	require.Equal(t, vfooV3, vfv3)
}

func TestEntityWrapperJSON(t *testing.T) {
	RegisterWrapper(&Foo{}, map[string]EntityI{
		DefaultOriginVersion: &foo{},
		"v2":                 &fooV2{},
		"v3":                 &fooV3{},
	})

	fv1 := foo{
		ID: "foo_id",
	}

	fooWp := &Foo{}
	fooWp.SetEntity(&fv1)

	dfoo, err := json.Marshal(fooWp)
	require.NoError(t, err)
	require.Equal(t, `{"ID":"foo_id"}`, string(dfoo))

	// unmarshal json data to fooWp
	ff := &Foo{}
	err = json.Unmarshal(dfoo, ff)
	require.NoError(t, err)
	require.Equal(t, fv1, *ff.Entity().(*foo))

	// data, err := fooWp.MarshalJSON()
	// require.NoError(t, err)

	fooWp2 := &Foo{}
	fv2 := &fooV2{
		Version: "v2",
		ID:      "foo_id",
		Name:    "foo_name",
	}

	fooWp2.SetEntity(fv2)

	dfoo2, err := json.Marshal(fooWp2)
	require.NoError(t, err)
	require.Equal(t, `{"Version":"v2","ID":"foo_id","Name":"foo_name"}`, string(dfoo2))

	require.NoError(t, json.Unmarshal(dfoo, fooWp2))

	vfv1, ok := fooWp2.Entity().(*foo)
	require.True(t, ok)
	require.Equal(t, "foo_id", vfv1.ID)

	ff2 := &Foo{}
	err = json.Unmarshal(dfoo2, ff2)
	require.NoError(t, err)
	require.Equal(t, fv2, ff2.Entity().(*fooV2))
}

func TestWrapperUpdateBase(t *testing.T) {
	RegisterWrapper(&Foo{}, map[string]EntityI{
		DefaultOriginVersion: &foo{},
		"v2":                 &fooV2{},
		"v3":                 &fooV3{},
	})

	fv1 := foo{
		ID: "foo_id",
	}

	fooWp := &Foo{}
	fooWp.SetEntity(&fv1)

	err := fooWp.UpdateBase(func(be EntityBaseI) error {
		be.(*fooBase).ID = "foo_id_v2"
		return nil
	})
	require.NoError(t, err)

	v, ok := fooWp.Entity().(*foo)
	require.True(t, ok)
	require.Equal(t, "foo_id_v2", v.ID)

	fv2 := fooV2{
		Version: "v2",
		ID:      "foo_id",
		Name:    "foo_name",
	}

	fooWp2 := &Foo{}
	fooWp2.SetEntity(&fv2)
	err = fooWp2.UpdateBase(func(be EntityBaseI) error {
		e, ok := be.(*fooBase)
		require.True(t, ok)
		e.ID = "foo_id_v2"
		return nil
	})
	require.NoError(t, err)
	v2, ok := fooWp2.Entity().(*fooV2)
	require.True(t, ok)
	require.Equal(t, "foo_id_v2", v2.ID)
	require.Equal(t, "foo_name", v2.Name)
	require.Equal(t, "v2", v2.Version)
}

func TestWrapperUpdateMigrate(t *testing.T) {
	RegisterWrapper(&Foo{}, map[string]EntityI{
		DefaultOriginVersion: &foo{},
		"v2":                 &fooV2{},
		"v3":                 &fooV3{},
	})

	fv1 := foo{
		ID: "foo_id",
	}

	fooWp := &Foo{}
	fooWp.SetEntity(&fv1)

	err := fooWp.Update(&fooV2{}, func(e EntityI) error {
		fv2 := e.(*fooV2)
		fv2.ID = "foo_id_v2"
		fv2.Name = "foo_new_name"
		return nil
	})
	require.NoError(t, err)

	fv2, ok := fooWp.Entity().(*fooV2)
	require.True(t, ok)
	require.Equal(t, "v2", fv2.Version)
	require.Equal(t, "foo_id_v2", fv2.ID)
	require.Equal(t, "foo_new_name", fv2.Name)
}
