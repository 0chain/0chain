package zcnsc_test

import (
	"0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScShouldHaveAddress(t *testing.T) {
	var sc = zcnsc.ZCNSmartContract{}
	address := sc.GetAddress()
	require.NotEmpty(t, address)
}
