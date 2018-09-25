package model

import (
	"strconv"

	"github.com/pmer/gobls"
)

type PartyId int

type Key gobls.SecretKey
type VerificationKey Key

type ThresholdError struct {
	By    PartyId
	Cause string
}

func (e ThresholdError) Error() string {
	return "Party " + strconv.Itoa(int(e.By)) + ": " + e.Cause
}
