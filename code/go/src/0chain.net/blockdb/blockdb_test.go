package blockdb

import (
	"fmt"
	"io"
	"testing"

	"0chain.net/common"
)

type Class struct {
	Grade       int8   `json:"grade"`
	Description string `json:"description"`
}

func (c *Class) Encode(writer io.Writer) error {
	buffer, err := common.ToMsgpack(c)
	if err != nil {
		return err
	}
	_, err = buffer.WriteTo(writer)
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
	buffer, err := common.ToMsgpack(c)
	if err != nil {
		return err
	}
	_, err = buffer.WriteTo(writer)
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
	compress := true
	db, err := NewBlockDB("/tmp/blockdb", 4, compress)
	if err != nil {
		panic(err)
	}
	err = db.Create()
	if err != nil {
		panic(err)
	}
	cls := &Class{Grade: 4, Description: "Most pouplar open source projects and technologies"}
	db.SetDBHeader(cls)
	students := make([]*Student, 3, 3)
	students[0] = &Student{Name: "Bitcoin - the first cryptocurrency", ID: "2009"}
	students[1] = &Student{Name: "Linux - the most popular open source operating system", ID: "1991"}
	students[2] = &Student{Name: "Apache - the first open source web server", ID: "1995"}
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

	db, err = NewBlockDB("/tmp/blockdb", 4, compress)
	if err != nil {
		panic(err)
	}
	db.SetDBHeader(cls2)
	err = db.Open()
	if err != nil {
		panic(err)
	}
	var sp StudentProvider
	srecs, err := db.ReadAll(&sp)
	if err != nil {
		panic(err)
	}
	for _, s := range srecs {
		fmt.Printf("student: %v\n", s)
	}
	db.Close()
}
