package indexer

type CurrentEpoch struct {
	Epoch int
}

type SlashingMultiplier struct {
	Multiplier string
}

type TargetApy struct {
	TargetApy string `json:"target_apy"`
}

type ElectedValidators struct {
	Validators []ElectedValidator
}

type ElectedValidator struct {
	Name    string
	Address string
	Group   string
}

type EpochVGRegistered struct {
	Block int
	Epoch int
}
