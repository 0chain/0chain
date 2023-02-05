package storagesc

import "sync"

type cache struct {
	config *Config
	l      sync.RWMutex
	err    error
}

var c = &cache{
	l: sync.RWMutex{},
}

func (*cache) update(conf *Config, err error) {
	c.l.Lock()
	c.config = conf
	c.err = err
	c.l.Unlock()
}
