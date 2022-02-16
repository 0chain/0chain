package utils

func GetCurrentRewardRound(currentRound, period int64) int64 {
	if period > 0 {
		extra := currentRound % period
		if currentRound >= extra {
			return currentRound - extra
		} else {
			return 0
		}
	}
	return 0
}

func GetPreviousRewardRound(currentRound, period int64) int64 {
	crr := GetCurrentRewardRound(currentRound, period)
	if crr >= period {
		return crr - period
	} else {
		return 0
	}
}
