package indexer

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/buidl-labs/celo-voting-validator-backend/graph/model"
	"github.com/go-pg/pg/v10"
)

func FindLastIndexedEpoch(DB *pg.DB) *model.Epoch {
	var epochs []*model.Epoch
	err := DB.Model(&epochs).Order("created_at desc").Limit(1).Select()
	if err != nil {
		log.Fatal(err)
	}
	return epochs[0]
}

func FindCurrentEpoch(client *http.Client) int {
	resp, err := client.Get("https://celo-on-chain-data-service.onrender.com/current-epoch")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	epoch := new(CurrentEpoch)
	json.NewDecoder(resp.Body).Decode(epoch)
	return epoch.Epoch
}

func GetVGSlashingMultiplier(client *http.Client, address string) string {
	resp, err := client.Get(fmt.Sprintf("https://celo-on-chain-data-service.onrender.com/downtime-score/%s", address))
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	multiplier := new(SlashingMultiplier)
	json.NewDecoder(resp.Body).Decode(multiplier)
	return multiplier.Multiplier
}

func GetTargetAPY(client *http.Client) string {
	resp, err := client.Get("https://celo-on-chain-data-service.onrender.com/target-apy")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	apy := new(TargetApy)
	json.NewDecoder(resp.Body).Decode(apy)
	return apy.TargetApy
}

func GetElectedValidators(client *http.Client) []ElectedValidator {
	resp, err := client.Get("https://celo-on-chain-data-service.onrender.com/elected-validators")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	electedValidators := new(ElectedValidators)
	json.NewDecoder(resp.Body).Decode(electedValidators)
	return electedValidators.Validators
}

func GetEpochVGRegistered(client *http.Client, address string) EpochVGRegistered {
	resp, err := client.Get(fmt.Sprintf("https://celo-on-chain-data-service.onrender.com/epoch-vg-registered/%s", address))
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	epochRegistered := new(EpochVGRegistered)
	json.NewDecoder(resp.Body).Decode(epochRegistered)
	return *epochRegistered
}
