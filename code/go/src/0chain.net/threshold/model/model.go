package model

import (
	"strconv"
)

type PartyId int

type Key [4]uint64
type VerificationKey Key

type ThresholdError struct {
	By    PartyId
	Cause string
}

func (e ThresholdError) Error() string {
	return "Party " + strconv.Itoa(int(e.By)) + ": " + e.Cause
}

const MyId = 0
