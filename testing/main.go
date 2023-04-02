package main

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/shruggr/1sat-indexer/lib"
)

var rdb *redis.Client

func main() {
	godotenv.Load("../.env")

	db, err := sql.Open("postgres", os.Getenv("POSTGRES"))
	if err != nil {
		log.Panic(err)
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	err = lib.Initialize(db, rdb)
	if err != nil {
		log.Panic(err)
	}
	txid, _ := hex.DecodeString("fe015378df4b217a476ac100d625cc79e7e901f39281431cdc8bb22fa8599ebb")
	tx, err := lib.LoadTx(txid)
	if err != nil {
		log.Panic(err)
	}

	result, err := lib.IndexTxos(tx, 0, 0, false)
	if err != nil {
		log.Panic(err)
	}
	out, err := json.Marshal(result)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(string(out))
}
