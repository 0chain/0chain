package blockdb

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"0chain.net/core/common"
)

type Class struct {
	Grade       int8   `json:"grade"`
	Description string `json:"description"`
}

func (c *Class) Encode(writer io.Writer) error {
	_, err := common.ToMsgpack(c).WriteTo(writer)
	return err
}

func (c *Class) Decode(reader io.Reader) error {
	return common.FromMsgpack(reader, c)
}

type Student struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func (s *Student) GetKey() Key {
	return Key(s.ID)
}

func (s *Student) Encode(writer io.Writer) error {
	_, err := common.ToMsgpack(s).WriteTo(writer)
	return err
}

func (s *Student) Decode(reader io.Reader) error {
	return common.FromMsgpack(reader, s)
}

type StudentProvider struct {
}

func (sp *StudentProvider) NewRecord() Record {
	return &Student{}
}

func TestDBWrite(t *testing.T) {
	db, err := NewBlockDB("/tmp/blockdb", 4, true)
	require.NoError(t, err)
	err = db.Create()
	require.NoError(t, err)

	cls := &Class{Grade: 4, Description: "Most pouplar open source projects and technologies"}
	db.SetDBHeader(cls)
	students := make([]*Student, 3, 3)
	students[0] = &Student{Name: "Bitcoin - the first cryptocurrency", ID: "2009"}
	students[1] = &Student{Name: "Linux - the most popular open source operating system", ID: "1991"}
	students[2] = &Student{Name: "Apache - the first open source web server", ID: "1995"}
	for _, s := range students {
		err = db.WriteData(s)
		require.NoError(t, err)
	}
	err = db.Save()
	require.NoError(t, err)
	cls2 := &Class{}
	db, err = NewBlockDB("/tmp/blockdb", 4, true)
	require.NoError(t, err)
	db.SetDBHeader(cls2)
	err = db.Open()
	require.NoError(t, err)
	for _, s := range students {
		var s2 Student
		err = db.Read(s.GetKey(), &s2)
		require.NoError(t, err)
	}
	err = db.Close()
	require.NoError(t, err)

	db, err = NewBlockDB("/tmp/blockdb", 4, true)
	require.NoError(t, err)
	db.SetDBHeader(cls2)
	err = db.Open()
	require.NoError(t, err)
	var sp StudentProvider
	_, err = db.ReadAll(&sp)
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}
