package entitywrapper

import (
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
		Name:    "foo_name",
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

//go:generate msgp -v -tests=false -io=false -unexported
// type foo struct {
// 	ID string `msg:"id"`
// }

// type fooV2 struct {
// 	Version string `msg:"version"`
// 	ID      string `msg:"id"`
// 	Name    string `msg:"name"`
// }

// type fooV3 struct {
// 	Version string `msg:"version"`
// 	Name    string `msg:"name"`
// 	Age     int    `msg:"age"`
// }
