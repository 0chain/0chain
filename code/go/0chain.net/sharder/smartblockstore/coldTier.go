package blockstore

type cloudStorageProvider interface {
	moveBlock()
	getBlock()
	getBlocks()
}

type minio struct {
	storageServiceURL string
	accessId          string
	secretAccessKey   string
	bucketName        string
}

func (ct *minio) moveBlock() {
	//
}
func (ct *minio) getBlock() {
	//
}
func (ct *minio) getBlocks() {
	//
}

type blobber struct {
	clientId      string
	clientKey     string
	allocationId  string
	allocationObj interface{} //put appropriate type later on

}

func (bl *blobber) moveBlock() {
	//
}

func (bl *blobber) getBlock() {
	//
}

func (bl *blobber) getBlocks() {
	//
}
