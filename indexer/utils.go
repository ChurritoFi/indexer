package indexer

import (
	"math"
	"math/big"
)

func divideStringByFloat(numerator string, denominator float64) float64 {
	num := convertStringToBigFloat(numerator)
	den := big.NewFloat(denominator)
	res := new(big.Float).Quo(num, den)
	resFloat, _ := res.Float64()
	return resFloat
}

func divideBy1E18(numerator string) float64 {
	OneE18 := float64(1e18)
	return divideStringByFloat(numerator, OneE18)
}

func divideBy1E24(numerator string) float64 {
	OneE24 := float64(1e24)
	return divideStringByFloat(numerator, OneE24)
}

func convertStringToBigFloat(number string) *big.Float {
	f, _, _ := big.ParseFloat(number, 10, 64, big.ToZero)
	return f
}

func getEpochFromBlock(block int) uint64 {
	if block == 0 {
		return 0
	}

	epochNumber := uint64(math.Floor(float64(block) / 17280.0))
	if block%17280 == 0 {
		return epochNumber
	} else {
		return epochNumber + 1
	}
}
