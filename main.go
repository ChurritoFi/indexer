package main

import (
	"context"
	"log"
	"os"

	"github.com/buidl-labs/celo-voting-validator-backend/graph/database"
	"github.com/buidl-labs/celo-voting-validator-backend/graph/model"
	"github.com/go-pg/pg/extra/pgdebug"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/joho/godotenv"
)

func main() {
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

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	DB_URL := os.Getenv("DB_URL")

	opts, err := pg.ParseURL(DB_URL)
	if err != nil {
		log.Fatal(err)
	}

	DB := database.New(opts)

	defer DB.Close()

	DB.AddQueryHook(pgdebug.DebugHook{
		Verbose: true,
	})

	ctx := context.Background()
	if err := DB.Ping(ctx); err != nil {
		log.Println(err)
	}
	DropAllTables(DB)
	CreateAllTables(DB)

	// var httpClient = &http.Client{Timeout: 10 * time.Second}
	// log.Println(indexer.FindCurrentEpoch(httpClient))
	// log.Println(indexer.GetVGSlashingMultiplier(httpClient, "0x8851F4852ce427191Dc8D9065d720619889e3260"))
	// log.Println(indexer.GetTargetAPY(httpClient))
	// log.Println(indexer.GetElectedValidators(httpClient))
	// log.Println(indexer.GetEpochVGRegistered(httpClient, "0x614B7654ba0cC6000ABe526779911b70C1F7125A"))
}

func DropAllTables(DB *pg.DB) {
	qs := []string{
		"drop table if exists epochs",
		"drop table if exists validators",
		"drop table if exists validator_stats",
		"drop table if exists validator_groups",
		"drop table if exists validator_group_stats",
	}

	for _, q := range qs {
		_, err := DB.Exec(q)
		if err != nil {
			panic(err)
		}
	}
}

func CreateAllTables(DB *pg.DB) {
	models := []interface{}{
		(*model.Epoch)(nil),
		(*model.ValidatorGroup)(nil),
		(*model.ValidatorGroupStats)(nil),
		(*model.Validator)(nil),
		(*model.ValidatorStats)(nil),
	}

	for _, model := range models {
		err := DB.Model(model).CreateTable(&orm.CreateTableOptions{
			Temp: false,
		})
		if err != nil {
			log.Print(err)
		}
	}
}
