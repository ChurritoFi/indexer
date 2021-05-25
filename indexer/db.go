package indexer

import (
	"github.com/buidl-labs/celo-voting-validator-backend/graph/model"
	"github.com/go-pg/pg/v10"
)

//NoResultError is thrown by go-pg when it can't find any result for the query
const NoResultError = "pg: no rows in result set"

func findLastIndexedEpoch(DB *pg.DB) (*model.Epoch, error) {

	epoch := new(model.Epoch)
	err := DB.Model(epoch).Order("created_at desc").Limit(1).Select()
	if err != nil {
		return epoch, err
	}

	return epoch, nil
}

func isVGInDB(DB *pg.DB, address string) (bool, error) {
	vg := new(model.ValidatorGroup)
	err := DB.Model(vg).Column("address").Where("address = ?", address).Select()
	if err.Error() == NoResultError {
		return false, err
	}

	return true, nil
}

func isVInDB(DB *pg.DB, address string) (bool, error) {
	v := new(model.Validator)
	err := DB.Model(v).Column("address").Where("address = ?", address).Select()
	if err.Error() == NoResultError {
		return false, err
	}

	return true, nil
}

// func findValidatorByAddress(DB *pg.DB, address string) (model.Validator, error) {
// 	var v model.Validator
// 	err := DB.Model(&v).Where("address = ?", address).Limit(1).Select()
// 	if err.Error() == NoResultError {
// 		return v, err
// 	}

// 	return v, nil
// }

// func findValidatorGroupByAddress(DB *pg.DB, address string) (model.ValidatorGroup, error) {
// 	var vg model.ValidatorGroup
// 	err := DB.Model(&vg).Where("address = ?", address).Limit(1).Select()
// 	if err.Error() == NoResultError {
// 		return vg, err
// 	}

// 	return vg, nil
// }
