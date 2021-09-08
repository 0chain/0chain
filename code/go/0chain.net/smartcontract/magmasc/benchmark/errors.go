package benchmark

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
