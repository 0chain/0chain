package cache

type Cache interface {
	New(size int)
	Add(key string, value interface{}) error
	Get(key string) (interface{}, error)
	GetHit() int64
	GetMiss() int64
}
