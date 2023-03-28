package cache

type Cache interface {
	Add(key string, value interface{}) error
	Get(key string) (interface{}, error)
	GetHit() int64
	GetMiss() int64
}
