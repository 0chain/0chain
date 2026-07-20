package state

import (
	"0dns.io/core/config"
	"0dns.io/core/logging"
	"github.com/0chain/gosdk/core/block"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGet(t *testing.T) {
	wantMiners := []string{"test1", "test2"}
	wantSharders := []string{"foo", "bar"}
	wantMagicBlock := &block.MagicBlock{
		Miners:   &block.NodePool{},
		Sharders: &block.NodePool{},
	}

	state.Miners = wantMiners
	state.Sharders = wantSharders
	state.CurrentMagicBlock = wantMagicBlock

	got := Get()

	assert.Equal(t, wantMiners, got.Miners)
	assert.Equal(t, wantSharders, got.Sharders)
	assert.Equal(t, wantMagicBlock, got.CurrentMagicBlock)
}

func TestSetFromCurrentMagicBlock(t *testing.T) {
	logging.InitLogging("development", os.TempDir())

	wantMiners := []string{"http://dev.0chain.net:10001", "http://dev.0chain.net:10002"}
	wantSharders := []string{"http://dev.0chain.net:20001", "http://dev.0chain.net:20002"}
	wantMagicBlock := &block.MagicBlock{
		Miners: &block.NodePool{
			Nodes: map[string]block.Node{
				"1": {
					Host: "dev.0chain.net",
					Port: 10001,
				},
				"2": {
					Host: "dev.0chain.net",
					Port: 10002,
				},
			},
		},
		Sharders: &block.NodePool{
			Nodes: map[string]block.Node{
				"1": {
					Host: "dev.0chain.net",
					Port: 20001,
				},
				"2": {
					Host: "dev.0chain.net",
					Port: 20002,
				},
			},
		},
	}

	SetFromCurrentMagicBlock(config.Config{}, wantMagicBlock)
	got := Get()

	assert.ElementsMatch(t, wantMiners, got.Miners)
	assert.ElementsMatch(t, wantSharders, got.Sharders)
	assert.Equal(t, wantMagicBlock, got.CurrentMagicBlock)
}
