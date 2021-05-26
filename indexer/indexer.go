package indexer

import (
	"log"
	"net/http"
	"time"

	"github.com/buidl-labs/celo-voting-validator-backend/graph/model"
	"github.com/go-pg/pg/v10"
	"github.com/machinebox/graphql"
)

//Index is a function that runs periodically to index the Celo chain.
func Index(DB *pg.DB) {

	/*
		### Indexing algorithm

		1. Find last indexed epoch.
		2. Find the current epoch.
		3. If the last indexed epoch is not the current epoch
			1. Index all the epochs between last indexed epoch and the current epoch.
			2. For each prev epoch, index only if the validator was elected or not(to calculate `epochs_served` for the VG)
		4. If last indexed epoch == current epoch
			1. Fetch all the elected validators for this epoch
			2. Fetch all the Validator Groups along with the data
			3. Make necessary API calls to the NodeJS service for getting on-chain data
			4. Calculate derived scores - performance score, transparency score etc.

	*/
	log.Println("Start indexing...")
	httpClient := &http.Client{Timeout: 30 * time.Second}
	gqlClient := graphql.NewClient("https://explorer.celo.org/graphiql")

	// Fetch all ValidatorGroups and Validators.

	vgData, err := getValidatorGroupsAndValidatorsBasicData(gqlClient)
	if err != nil {
		log.Println("Couldn't fetch data.")
		log.Println(err.Error())
	}
	log.Println("Fetched all VGs")

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
	log.Println("Finished indexing VGs and Vs.")

	// Start indexing elected validators for the past epoch.
	var epochToIndexFrom uint64
	lastIndexedEpoch, err := findLastIndexedEpoch(DB)

	if err != nil {
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
		return
	}
	log.Println("Current epoch:", currentEpoch)

	if epochToIndexFrom != currentEpoch {

		// Index the elected validators of all the prev epoch.
		/*
			1. Fetch all VGs from DB.
			2. Loop through the validators on every epoch
			3. Increase epochs served for the vg, once per epoch. Break the loop if found the group once in the epoch.
			4. Make the changes to the DB
		*/

		var validatorGroupsFromDB []*model.ValidatorGroup
		if err := DB.Model(&validatorGroupsFromDB).Select(); err != nil {
			log.Fatal(err)
		}
		for epoch := epochToIndexFrom; epoch < currentEpoch; epoch++ {
			vgInEpoch := make(map[string]bool)

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
				return
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

			for _, v := range electedValidatorsInEpoch.CeloElectedValidators {
				if v.CeloAccount.Validator.GroupInfo.Address != "" {
					vgInEpoch[v.CeloAccount.Validator.GroupInfo.Address] = true
				}
			}

			for vg, served := range vgInEpoch {
				if served {
					vgFromDB := new(model.ValidatorGroup)
					found := false
					for _, groupFromDB := range validatorGroupsFromDB {
						if vg == groupFromDB.Address {
							vgFromDB = groupFromDB
							found = true
							break
						}
					}
					if found {
						vgFromDB.EpochsServed++
						_, err := DB.Model(vgFromDB).WherePK().Update()
						if err != nil {
							log.Fatalln(err)
						}
					}
				}
			}
			time.Sleep(2 * time.Second)
		}

	}

	// Index the current epoch.
	log.Println("Index the current epoch")

	// Fetch currently elected validators.
	// currentlyElectedValidators := getElectedValidators(httpClient)
	details, err := getValidatorGroupsAndValidatorsDetails(gqlClient)
	log.Println(len(details.CeloValidatorGroups))

}
