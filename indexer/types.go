package indexer

type currentEpoch struct {
	Epoch uint64
}

type slashingMultiplier struct {
	Multiplier string
}

type targetApy struct {
	TargetApy string `json:"target_apy"`
}

type electedValidators struct {
	Validators []electedValidator
}

type electedValidator struct {
	Name    string
	Address string
	Group   string
}

type epochVGRegistered struct {
	Block int
	Epoch int
}

type validatorGroupAndValidatorsBasicData struct {
	CeloValidatorGroups []celoValidatorGroupAndValidatorBasicData `json:"celoValidatorGroups"`
}

type celoValidatorGroupAndValidatorBasicData struct {
	Account struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	} `json:"account"`
	Affiliates struct {
		Edges []struct {
			Node struct {
				Address string `json:"address"`
				Name    string `json:"name"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"affiliates"`
}

type electedValidatorsAtEpoch struct {
	CeloElectedValidators []struct {
		CeloAccount struct {
			Address   string `json:"address"`
			Validator struct {
				GroupInfo struct {
					Address string `json:"address"`
				} `json:"groupInfo"`
			} `json:"validator"`
		} `json:"celoAccount"`
	} `json:"celoElectedValidators"`
}
