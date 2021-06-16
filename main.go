package main

import (
	"context"
	"log"
	"os"

	"github.com/buidl-labs/celo-indexer/indexer"
	"github.com/buidl-labs/celo-voting-validator-backend/graph/database"
	"github.com/buidl-labs/celo-voting-validator-backend/graph/model"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	DbURL := os.Getenv("DB_URL")

	opts, err := pg.ParseURL(DbURL)
	if err != nil {
		log.Fatal(err)
	}

	DB := database.New(opts)

	defer DB.Close()

	// DB.AddQueryHook(pgdebug.DebugHook{
	// 	Verbose: true,
	// })

	ctx := context.Background()
	if err := DB.Ping(ctx); err != nil {
		log.Println(err)
	}
	// dropAllTables(DB)
	// createAllTables(DB)

	indexer.Index(DB)

}

func dropAllTables(DB *pg.DB) {
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

func createAllTables(DB *pg.DB) {
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
