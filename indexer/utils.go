package indexer

import (
	"math"
	"math/big"

	"github.com/buidl-labs/celo-voting-validator-backend/graph/model"
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

func calculateCeloPerValidator(celo uint64, num_validators uint) float64 {
	if celo == 0 || num_validators == 0 {
		return 0
	}
	return float64(celo) / float64(num_validators)
}

func calculateTransparencyScore(vg *model.ValidatorGroup) float64 {
	transparencyScore := float64(0)
	if vg.WebsiteURL != "" {
		transparencyScore += 0.15
		if vg.VerifiedDNS {
			transparencyScore += 0.25
		}
	}
	if vg.Name != "" {
		transparencyScore += 0.15
	}
	if vg.Email != "" {
		transparencyScore += 0.15
	}
	if vg.GeographicLocation != "" {
		transparencyScore += 0.1
	}
	if vg.TwitterUsername != "" {
		transparencyScore += 0.1
	}
	if vg.DiscordTag != "" {
		transparencyScore += 0.1
	}

	return transparencyScore
}

func calculatePerformanceScore(vg *model.ValidatorGroup, totalEpochs float64) float64 {
	/*
		SlashingMultiplier(30%)
		Group Score(30%)
		EpochsServedHistory(10%)
		EpochsServedCapacity(10%)
		LockedceloPercentile(6%)
		Number of Elected Validators(2%) -> 0.04 * Number of Elected Validators
		Percentage of Validators Elected(6%) -> (elected validators / total validators) * 0.04
		Attestation Score(6%)
	*/

	thirtyPercent := 0.3
	tenPercent := 0.1
	sixPercent := 0.06
	twoPercent := 0.02
	ZeroPointFourPercent := twoPercent / 5.0

	performanceScore := float64(0)
	performanceScore += (vg.SlashingPenaltyScore * thirtyPercent)
	performanceScore += (vg.GroupScore * thirtyPercent)

	epochsServedHistoryPercent := float64(vg.EpochsServed) / float64(totalEpochs)
	epochsAvailable := totalEpochs - float64(vg.EpochRegisteredAt)
	var epochsServedHistoryCapacity float64
	if epochsAvailable == 0 {
		epochsServedHistoryCapacity = 0.0
	} else {
		epochsServedHistoryCapacity = math.Min((float64(vg.EpochsServed) / epochsAvailable), float64(1))
	}

	performanceScore += (epochsServedHistoryPercent * tenPercent)
	performanceScore += (epochsServedHistoryCapacity * tenPercent)

	performanceScore += (vg.LockedCeloPercentile * sixPercent)

	numElectedValidators := 0
	totalValidators := 0
	for _, v := range vg.Validators {
		if v.CurrentlyElected {
			numElectedValidators++
		}
		totalValidators++
	}
	if totalValidators > 0 {
		performanceScore += ((float64(numElectedValidators) / float64(totalValidators)) * sixPercent)
	}
	performanceScore += (vg.AttestationScore * sixPercent)

	performanceScore += (float64(numElectedValidators) * ZeroPointFourPercent)

	return performanceScore
}
