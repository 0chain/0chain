package vestingsc

import "sync"

type cache struct {
	config *config
	l      sync.RWMutex
	err    error
}

var cfg = &cache{
	l: sync.RWMutex{},
}

func (*cache) update(conf *config, err error) {
	cfg.l.Lock()
	cfg.config = conf
	cfg.err = err
	cfg.l.Unlock()
}
