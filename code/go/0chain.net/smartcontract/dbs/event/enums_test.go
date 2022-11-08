package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventTags(t *testing.T) {
	require.Len(t, TagString, NumberOfTags.Int())
}
