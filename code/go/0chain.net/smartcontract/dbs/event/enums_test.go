package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventType(t *testing.T) {
	require.Len(t, TypeSting, TypeStats.Int())
}

func TestEventTags(t *testing.T) {
	require.Len(t, TagString, NumberOfTags.Int())
}
