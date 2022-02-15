package utils

func GetCurrentRewardRound(currentRound, period int64) int64 {
	extra := currentRound % period
	return currentRound - extra
}

func GetPreviousRewardRound(currentRound, period int64) int64 {
	crr := GetCurrentRewardRound(currentRound, period)
	return crr - period
}
