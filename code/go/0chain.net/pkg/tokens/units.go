package tokens

//go:generate msgp -io=false -tests=false -v
//Balance - any quantity that is represented as an integer in the lowest denomination
type Balance int64
