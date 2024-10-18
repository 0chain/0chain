package minersc

import (
	"errors"

	"0chain.net/core/util/entitywrapper"
)

//msgp:ignore GlobalNode
//go:generate msgp -io=false -tests=false -unexported -v

const globalNodeV2Version = "v2"

func init() {
	entitywrapper.RegisterWrapper(&GlobalNode{},
		map[string]entitywrapper.EntityI{
			entitywrapper.DefaultOriginVersion: &globalNodeV1{},
			"v2":                               &globalNodeV2{},
		})
}

type globalNodeV2 struct {
	globalNodeV1
	Version       string `msg:"version"`
	Name          string `msg:"name"`
	VCPhaseRounds []int  `msg:"vc_phase_rounds"`
}

func (gn2 *globalNodeV2) GetVersion() string {
	return globalNodeV2Version
}

func (gn2 *globalNodeV2) InitVersion() {
	gn2.Version = globalNodeV2Version
}

func (gn2 *globalNodeV2) GetBase() entitywrapper.EntityBaseI {
	b := globalNodeBase(gn2.globalNodeV1)
	return &b
}

func (gn2 *globalNodeV2) MigrateFrom(e entitywrapper.EntityI) error {
	v1, ok := e.(*globalNodeV1)
	if !ok {
		return errors.New("struct migrate fail, wrong global node type")
	}

	gn2.ApplyBaseChanges(globalNodeBase(*v1))
	gn2.Version = globalNodeV2Version
	return nil
}

func (gn2 *globalNodeV2) ApplyBaseChanges(gnb globalNodeBase) {
	gn2.globalNodeV1 = globalNodeV1(gnb)
}
