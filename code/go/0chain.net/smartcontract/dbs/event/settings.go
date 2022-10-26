package event

import "0chain.net/chaincore/config"

type Settings struct {
	Debug bool
}

func newSettings(config config.DbAccess) *Settings {
	return &Settings{
		Debug: config.Debug,
	}
}
