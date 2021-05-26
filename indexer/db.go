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
