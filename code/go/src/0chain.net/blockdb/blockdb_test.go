package blockdb

import (
	"fmt"
	"io"
	"testing"

	"0chain.net/common"
)

type Class struct {
	Grade int8 `json:"grade"`
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

func TestDBWrite(t *testing.T) {
	compress := false
	db, err := NewBlockDB("/tmp/blockdb", 4, compress)
	if err != nil {
		panic(err)
	}
	err = db.Create()
	if err != nil {
		panic(err)
	}
	cls := &Class{Grade: 4}
	db.SetDBHeader(cls)
	students := make([]*Student, 3, 3)
	students[0] = &Student{Name: "Bitcoin", ID: "2009"}
	students[1] = &Student{Name: "Linux", ID: "1991"}
	students[2] = &Student{Name: "Apache", ID: "1995"}
	for _, s := range students {
		err = db.WriteData(s)
		if err != nil {
			panic(err)
		}
	}
	err = db.Save()
	if err != nil {
		panic(err)
	}
	cls2 := &Class{}
	db, err = NewBlockDB("/tmp/blockdb", 4, compress)
	if err != nil {
		panic(err)
	}
	db.SetDBHeader(cls2)
	err = db.Open()
	if err != nil {
		panic(err)
	}
	fmt.Printf("class: %v\n", cls2)
	for _, s := range students {
		var s2 Student
		fmt.Printf("reading the key: %v\n", s.GetKey())
		err = db.Read(s.GetKey(), &s2)
		if err != nil {
			panic(err)
		}
		fmt.Printf("student: %v\n", s2)
	}
	db.Close()
}
