package model

type ID int

type KeyShare interface{}

type Sign interface{}

type SimpleDKG interface {

func GetKeyShareForOther(to ID) KeyShare
func ReceiveKeyShare(from ID, share KeyShare) error

}

