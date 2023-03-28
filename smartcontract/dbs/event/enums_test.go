package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventType(t *testing.T) {
	initTypeString()
	require.Len(t, TypeString, NumberOfTypes.Int()+1)
}

func TestEventTags(t *testing.T) {
	initTagString()
	require.Len(t, TagString, NumberOfTags.Int()+1)
}
