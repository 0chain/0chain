package block

import (
	"testing"
)

func TestLastBlockEvents(t *testing.T) {
	SetupBlockEventDB("./test_data/")
	SetupBlockEventEntity()
}
