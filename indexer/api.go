package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/machinebox/graphql"
)

func findCurrentEpoch(client *http.Client) (uint64, error) {
	resp, err := client.Get("https://celo-on-chain-data-service.onrender.com/current-epoch")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	epoch := new(currentEpoch)
	json.NewDecoder(resp.Body).Decode(epoch)
	return epoch.Epoch, nil
}

func getVGSlashingMultiplier(client *http.Client, address string) (string, error) {
	resp, err := client.Get(fmt.Sprintf("https://celo-on-chain-data-service.onrender.com/downtime-score/%s", address))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	multiplier := new(slashingMultiplier)
	json.NewDecoder(resp.Body).Decode(multiplier)
	return multiplier.Multiplier, nil
}

func getTargetAPY(client *http.Client) (string, error) {
	resp, err := client.Get("https://celo-on-chain-data-service.onrender.com/target-apy")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	apy := new(targetApy)
	json.NewDecoder(resp.Body).Decode(apy)
	return apy.TargetApy, nil
}

func getElectedValidators(client *http.Client) ([]electedValidator, error) {
	electedValidators := new(electedValidators)

	resp, err := client.Get("https://celo-on-chain-data-service.onrender.com/elected-validators")
	if err != nil {
		return electedValidators.Validators, err
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(electedValidators)
	return electedValidators.Validators, nil
}

func getEpochVGRegistered(client *http.Client, address string) (epochVGRegistered, error) {
	epochRegistered := new(epochVGRegistered)
	resp, err := client.Get(fmt.Sprintf("https://celo-on-chain-data-service.onrender.com/epoch-vg-registered/%s", address))
	if err != nil {
		return *epochRegistered, err
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(epochRegistered)
	return *epochRegistered, nil
}

func getValidatorGroupsAndValidatorsBasicData(client *graphql.Client) (validatorGroupAndValidatorsBasicData, error) {
	req := graphql.NewRequest(`{
	  celoValidatorGroups{
	    account{
	      address
	      name
	    }
	    affiliates(first: 10){ 
	      edges{
	        node{
	          name
	          address
	        }
	      }
			}
	  }  
	}`)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))
	defer cancel()
	var resp validatorGroupAndValidatorsBasicData
	if err := client.Run(ctx, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func getElectedValidatorsAtEpoch(client *graphql.Client, epoch uint64) (electedValidatorsAtEpoch, error) {
	const BlocksPerEpoch = 17280
	req := graphql.NewRequest(`
		query($block: Int!){ 
			celoElectedValidators(blockNumber: $block) { 
				celoAccount{
					address
					validator{
						groupInfo{
							address
						}
					}
				}				                                              	
			}
		}
	`)
	var blockNumber uint64
	var resp electedValidatorsAtEpoch
	if epoch < 1 {
		return resp, errors.New("error: epoch needs to be greater than or equal to 1")
	} else if epoch == 1 {
		blockNumber = BlocksPerEpoch / 2
	} else {
		blockNumber = ((epoch - 1) * BlocksPerEpoch) + 500
	}
	req.Var("block", blockNumber)
	log.Println("Finding elected validators at block:", blockNumber)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))
	defer cancel()
	if err := client.Run(ctx, req, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

func getValidatorGroupsAndValidatorsDetails(client *graphql.Client) (celoValidatorGroupsAndValidatorsDetails, error) {
	req := graphql.NewRequest(`{
			celoValidatorGroups {
				account {
					address
					name
					group {
						commission
						lockedGold
						receivableVotes
						votes
					}
					claims(first: 10){
						edges{
							node {
								element
								type
								verified
							}
						}
					}
		
				}
				numMembers
				affiliates(first: 5) {
					edges {
						node {
							lastElected
							score
							address
							attestationsFulfilled
							attestationsRequested
							score
						}
					}
				}
				accumulatedRewards
				accumulatedActive
			}
		}
		`)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*90))
	defer cancel()
	var resp celoValidatorGroupsAndValidatorsDetails
	if err := client.Run(ctx, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}
