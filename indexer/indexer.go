package indexer

import (
	"log"
	"math"
	"math/big"
	"net/http"
	"time"

	"github.com/buidl-labs/celo-voting-validator-backend/graph/model"
	"github.com/go-pg/pg/v10"
	"github.com/machinebox/graphql"
)

//Index is a function that runs periodically to index the Celo chain.
func Index(DB *pg.DB) {

	log.Println("Start indexing...")

	// Clients used to fetch data from APIs.
	httpClient := &http.Client{Timeout: 30 * time.Second}
	gqlClient := graphql.NewClient("https://explorer.celo.org/graphiql")

	// Fetch all ValidatorGroups and Validators.
	vgData, err := getValidatorGroupsAndValidatorsBasicData(gqlClient)
	if err != nil {
		log.Println("Couldn't fetch data.")
		log.Println(err)
	} else {
		log.Println("Fetched all VGs")
	}

	// Loop through all the ValidatorGroups
	for _, vg := range vgData.CeloValidatorGroups {

		// Check if VG is in DB
		// Potential Improvement: Fetch all VGs from the DB at once, and then cross-check against that list.
		vgFromDB := new(model.ValidatorGroup)
		err := DB.Model(vgFromDB).Where("address = ?", vg.Account.Address).Limit(1).Select()

		if err != nil {

			// If VG isn't in DB -> Add the VG to DB.
			if err.Error() == NoResultError {

				// Fetch the epoch VG was registered at.
				epochRegistered, err := getEpochVGRegistered(httpClient, vg.Account.Address)
				if err != nil {
					log.Println(err)
					return
				}

				vgForDB := model.ValidatorGroup{
					Address:           vg.Account.Address,
					Name:              vg.Account.Name,
					EpochRegisteredAt: uint64(epochRegistered.Epoch),
				}

				_, err = DB.Model(&vgForDB).Insert()
				if err != nil {
					log.Println(err)
					return
				}

				// Loop through the Validators of the ValidatorGroup
				// Potential Improvement: Remove validators from the group that have de-registered.
				for _, v := range vg.Affiliates.Edges {
					// Check if Validator is in DB
					vFromDB := new(model.Validator)
					err := DB.Model(vFromDB).Where("address = ?", v.Node.Address).Limit(1).Select()

					// If Validator isn't in DB; Insert it into the DB.
					if err.Error() == NoResultError {
						vForDB := model.Validator{
							Address:          v.Node.Address,
							Name:             v.Node.Name,
							ValidatorGroupId: vgForDB.ID,
						}
						_, err := DB.Model(&vForDB).Insert()
						if err != nil {
							return
						}
					}
				}
			} else {
				log.Panic(err.Error())
				return
			}
		}
	} // Finished indexing new ValidatorGroups, and Validators.
	log.Println("Finished looping through VGs and Vs.")

	var epochToIndexFrom uint64
	lastIndexedEpoch, err := findLastIndexedEpoch(DB)

	if err != nil {
		// If no Epochs are present in DB, start from Epoch 1.
		if err.Error() == NoResultError {
			epochToIndexFrom = 1
		}
	} else {
		epochToIndexFrom = lastIndexedEpoch.Number + 1
	}

	log.Println("Epoch to index from:", epochToIndexFrom)

	currentEpoch, err := findCurrentEpoch(httpClient)
	if err != nil {
		log.Fatal("Error fetching current epoch.")
	}
	log.Println("Current epoch:", currentEpoch)

	// Index prev epochs if epochToIndexFrom != currentEpoch
	if epochToIndexFrom != currentEpoch {
		var validatorGroupsFromDB []*model.ValidatorGroup
		if err := DB.Model(&validatorGroupsFromDB).Select(); err != nil {
			log.Fatal(err)
		}

		// Loop through all the epochs between epochToIndexFrom - currentEpoch
		for epoch := epochToIndexFrom; epoch < currentEpoch; epoch++ {

			// Keeps tally of which VGs have served in the Epoch.
			vgInEpoch := make(map[string]bool)

			// Used to inserting the Epoch to DB.
			var startBlock, endBlock uint64

			if epoch == 1 {
				startBlock = 1
				endBlock = 17280
			} else {
				startBlock = ((epoch - 1) * 17280) + 1
				endBlock = (epoch * 17280)
			}

			log.Println("For epoch", epoch)
			electedValidatorsInEpoch, err := getElectedValidatorsAtEpoch(gqlClient, epoch)
			if err != nil {
				log.Println("Error fetching elected validators.")
				log.Fatal(err.Error())
			}

			currEpoch := model.Epoch{
				StartBlock: startBlock,
				EndBlock:   endBlock,
				Number:     epoch,
			}
			_, err = DB.Model(&currEpoch).Insert()
			if err != nil {
				log.Fatal(err)
				return
			}

			// Aggregate the VGs that were elected in this Epoch.
			for _, v := range electedValidatorsInEpoch.CeloElectedValidators {
				if v.CeloAccount.Validator.GroupInfo.Address != "" {
					vgInEpoch[v.CeloAccount.Validator.GroupInfo.Address] = true
				}
			}

			// Loop through all the VGs that have served in this epoch
			for vg, served := range vgInEpoch {

				if served {

					found := false // keeps track of whether the vg we're looking for is present in the DB or not.
					vgFromDB := new(model.ValidatorGroup)
					// Loops through VGs from DB to find the VG we want to update.
					for _, groupFromDB := range validatorGroupsFromDB {
						if vg == groupFromDB.Address {
							vgFromDB = groupFromDB
							found = true
							break
						}
					}

					if found {
						// Increment the EpochsServed and update the DB.
						vgFromDB.EpochsServed++
						_, err := DB.Model(vgFromDB).WherePK().Update()
						if err != nil {
							log.Fatalln(err)
						}
					}
				}
			}

			// Small pause to not overload the API we're using to fetch the ElectedValidators
			time.Sleep(3 * time.Second)
		}

	}

	// Index the current epoch.
	log.Println("Index the current epoch")

	// `isCurrentEpochIndexedBefore` used to check whether we need to increment `EpochsServed` for the VG.
	// Intent is to only increment `EpochsServed` for the VG if it's the first time we're indexing the epoch.
	isCurrentEpochIndexedBefore := true

	latestEpoch := new(model.Epoch)
	// Find the model.Epoch from DB for the current epoch.
	err = DB.Model(latestEpoch).Where("number = ?", currentEpoch).Limit(1).Select()
	log.Println("Epoch number:", latestEpoch.Number)

	if err != nil {
		if err.Error() == NoResultError {
			// If current epoch isn't present in the DB, insert it into the DB.
			isCurrentEpochIndexedBefore = false
			latestEpoch = &model.Epoch{
				StartBlock: ((currentEpoch - 1) * 17280) + 1,
				EndBlock:   currentEpoch * 17280,
				Number:     currentEpoch,
			}

			_, err = DB.Model(latestEpoch).Insert()
			if err != nil {
				log.Fatal(err)
				return
			}
		}
	}

	// target yield is the parameter set by the Celo network to adjust inflation schedule
	targetYield, _ := getTargetAPY(httpClient)
	targetYieldFloat := convertStringToBigFloat(targetYield)
	log.Printf("%f target apy", targetYieldFloat)

	details, err := getValidatorGroupsAndValidatorsDetails(gqlClient)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch all the VGs and Vs from the DB.
	var validatorGroupsFromDB []*model.ValidatorGroup
	err = DB.Model(&validatorGroupsFromDB).Relation("Validators").Select()
	if err != nil {
		log.Println(err)
	}

	// Loop through all the ValidatorGroups
	for _, validatorGroup := range details.CeloValidatorGroups {

		// Find the current vg from the validatorGroupsFromDB
		vgFromDB := new(model.ValidatorGroup)
		for _, vg := range validatorGroupsFromDB {
			if vg.Address == validatorGroup.Account.Address {
				vgFromDB = vg
				break
			}
		}
		log.Printf("%s(%s)\n", vgFromDB.Name, vgFromDB.Address)

		isVGCurrentlyElected := false               // Used for updating VG
		validatorScores := make([]float64, 0, 10)   // Used for calculating `GroupScore` for the VG
		attestationScores := make([]float64, 0, 10) // Used for calculating `AttestationPercentage` for the VG

		// Loop through the Validators in the ValidatorGroup
		for _, validator := range validatorGroup.Affiliates.Edges {

			// Find the current validator from DB
			vFromDB := new(model.Validator)
			for _, v := range vgFromDB.Validators {
				if v.Address == validator.Node.Address {
					vFromDB = v
				}
			}

			vScore := divideBy1E24(validator.Node.Score)

			// Current round of stats for the Validator
			vStats := &model.ValidatorStats{
				AttestationsRequested: validator.Node.AttestationsRequested,
				AttestationsFulfilled: validator.Node.AttestationsFulfilled,
				LastElected:           validator.Node.LastElected,
				Score:                 vScore,
				EpochId:               lastIndexedEpoch.ID,
				ValidatorId:           vFromDB.ID,
			}
			_, err := DB.Model(vStats).Insert()

			if err != nil {
				log.Fatal(err)
			}

			// Find which is the epoch, validator was last elected in.
			epochLastElected := getEpochFromBlock(validator.Node.LastElected)

			if epochLastElected == currentEpoch {
				vFromDB.CurrentlyElected = true
				isVGCurrentlyElected = true

				// Do this inside this IF branch because, only consider validator scores of elected validators for the `GroupScore`
				validatorScores = append(validatorScores, vScore)
			}

			// Used for calculating `AttestationPercentage` for the VG.
			if vStats.AttestationsFulfilled > 0 {
				attestationScores = append(attestationScores, (float64(vStats.AttestationsFulfilled) / float64(vStats.AttestationsRequested)))
			}

			_, err = DB.Model(vFromDB).WherePK().Update()
			if err != nil {
				log.Fatal(err)
			}

		} // Finish indexing Validators under the ValidatorGroup

		// If VG is currently elected, increment VG.EpochsServed
		if isVGCurrentlyElected && !isCurrentEpochIndexedBefore {
			vgFromDB.EpochsServed++
		}

		// Loop through the claims, set VG.WebsiteURL if claim is of type "domain"
		for _, claim := range validatorGroup.Account.Claims.Edges {
			if claim.Node.Type == "domain" {
				vgFromDB.WebsiteURL = claim.Node.Element
				vgFromDB.VerifiedDNS = claim.Node.Verified
			}
		}

		lockedCelo := uint64(divideBy1E18(validatorGroup.Account.Group.LockedGold))
		groupShare := divideBy1E24(validatorGroup.Account.Group.Commission)
		votes := uint64(divideBy1E18(validatorGroup.Account.Group.Votes))
		// votingCap = votesRecieved + availableVotes
		votingCap := votes + uint64(divideBy1E18(validatorGroup.Account.Group.ReceivableVotes))
		slashingScore, _ := getVGSlashingMultiplier(httpClient, validatorGroup.Account.Address)
		slashingScoreFloat := float64(0)
		if slashingScore != "" {
			slashingScoreFloat = divideBy1E24(slashingScore)
		}

		// groupScore is the average of all elected validators under the VG
		groupScore := float64(0)
		if len(validatorScores) > 0 {
			for _, vScore := range validatorScores {
				groupScore += vScore
			}
			groupScore /= float64(len(validatorScores))
		}
		// estimatedAPY = targetAPY * groupScore
		estimatedAPYFloat := float64(0)
		if groupScore > 0 {
			estimatedAPY := new(big.Float).Mul(targetYieldFloat, big.NewFloat(groupScore))
			estimatedAPYFloat, _ = estimatedAPY.Float64()
		}

		// groupAttestationScore(`AttestationPercentage`) is the average of the attestation scores(attestations requested / attestations fulfilled) of each Validator
		groupAttestationScore := float64(0)
		if len(attestationScores) > 0 {
			for _, attestationScore := range attestationScores {
				groupAttestationScore += attestationScore
			}
			groupAttestationScore /= float64(len(attestationScores))
		}

		groupTransparencyScore := calculateTransparencyScore(vgFromDB)

		// Current round of VGStats for the VG
		vgStats := &model.ValidatorGroupStats{
			LockedCelo:            lockedCelo,
			GroupShare:            groupShare,
			Votes:                 votes,
			VotingCap:             votingCap,
			AttestationPercentage: groupAttestationScore,
			SlashingScore:         slashingScoreFloat,
			EpochId:               latestEpoch.ID,
			ValidatorGroupId:      vgFromDB.ID,
			EstimatedAPY:          estimatedAPYFloat,
		}

		// Update the current stats for the VG
		vgFromDB.AvailableVotes = vgStats.VotingCap - vgStats.Votes
		vgFromDB.RecievedVotes = vgStats.Votes
		vgFromDB.CurrentlyElected = isVGCurrentlyElected
		vgFromDB.LockedCelo = uint64(vgStats.LockedCelo)
		vgFromDB.SlashingPenaltyScore = slashingScoreFloat
		vgFromDB.GroupScore = groupScore
		vgFromDB.AttestationScore = groupAttestationScore
		vgFromDB.EstimatedAPY = estimatedAPYFloat
		vgFromDB.TransparencyScore = groupTransparencyScore
		vgFromDB.GroupShare = groupShare

		// Insert VGStats for the current round.
		_, err = DB.Model(vgStats).Insert()
		if err != nil {
			log.Println(err)
		}

		// Update vgFromDB in the DB.
		_, err = DB.Model(vgFromDB).WherePK().Update()
		if err != nil {
			log.Fatal(err)
		}

	}

	// Calculate LockedCelo / NumValidators per VG
	lockedCeloByNumValidatorsPerVG := make(map[string]float64)
	maxLockedCeloByNumValidators := math.Inf(-1)

	for _, vg := range validatorGroupsFromDB {
		val := calculateCeloPerValidator(vg.LockedCelo, uint(len(vg.Validators)))
		lockedCeloByNumValidatorsPerVG[vg.Address] = val
		maxLockedCeloByNumValidators = math.Max(val, maxLockedCeloByNumValidators)
	}

	// Calculate (LockedCelo/NumValidators)Percentile and Performance Score for each VG.
	for _, vg := range validatorGroupsFromDB {
		VGLockedCeloByNumValidators, ok := lockedCeloByNumValidatorsPerVG[vg.Address]
		if !ok {
			continue
		}

		vg.LockedCeloPercentile = VGLockedCeloByNumValidators / maxLockedCeloByNumValidators

		vgPerformanceScore := calculatePerformanceScore(vg, float64(currentEpoch))
		vg.PerformanceScore = vgPerformanceScore

		_, err := DB.Model(vg).WherePK().Update()
		if err != nil {
			log.Fatal("Couldn't update VG.")
		}
	}

}
