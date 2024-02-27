package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/shruggr/1sat-indexer/indexer"
	"github.com/shruggr/1sat-indexer/lib"
	"github.com/shruggr/1sat-indexer/ordinals"
)

var POSTGRES string
var db *pgxpool.Pool
var rdb *redis.Client
var INDEXER string
var TOPIC string
var FROM_BLOCK uint
var VERBOSE int
var CONCURRENCY int = 64

func init() {
	wd, _ := os.Getwd()
	log.Println("CWD:", wd)
	godotenv.Load(fmt.Sprintf(`%s/../../.env`, wd))

	flag.StringVar(&INDEXER, "id", "", "Indexer name")
	flag.StringVar(&TOPIC, "t", "", "Junglebus SubscriptionID")
	flag.UintVar(&FROM_BLOCK, "s", uint(lib.TRIGGER), "Start from block")
	flag.IntVar(&CONCURRENCY, "c", 64, "Concurrency Limit")
	flag.IntVar(&VERBOSE, "v", 0, "Verbose")
	flag.Parse()

	if POSTGRES == "" {
		POSTGRES = os.Getenv("POSTGRES_FULL")
	}
	var err error
	log.Println("POSTGRES:", POSTGRES)
	db, err = pgxpool.New(context.Background(), POSTGRES)
	if err != nil {
		log.Panic(err)
	}

	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}

	rdb = redis.NewClient(opts)

	err = indexer.Initialize(db, rdb)
	if err != nil {
		log.Panic(err)
	}

}

func main() {

	err := indexer.Exec(
		true,
		false,
		func(ctx *lib.IndexContext) error {
			ordinals.ParseInscriptions(ctx)
			ids := map[string]uint64{}
			for _, txo := range ctx.Txos {
				if bsv20, ok := txo.Data["bsv20"].(*ordinals.Bsv20); ok {
					if bsv20.Id == nil {
						continue
					}
					id := bsv20.Id.String()
					if txouts, ok := ids[id]; !ok {
						ids[id] = 1
					} else {
						ids[id] = txouts + 1
					}
				}
			}
			for idstr, txouts := range ids {
				id, err := lib.NewOutpointFromString(idstr)
				if err != nil {
					log.Printf("Err: %s %x %d\n", idstr, ctx.Txid, txouts)
					return err
				} else {
					_, err = db.Exec(context.Background(), `
						INSERT INTO bsv20v2_txns(txid, id, height, idx, txouts)
						VALUES($1, $2, $3, $4, $5)
						ON CONFLICT(txid, id) DO NOTHING`,
						ctx.Txid,
						id,
						ctx.Height,
						ctx.Idx,
						txouts,
					)
				}
				if err != nil {
					log.Printf("Err: %s %x %d\n", idstr, ctx.Txid, txouts)
					return err
				}
			}
			return nil
		},
		func(height uint32) error {

			return nil
		},
		INDEXER,
		TOPIC,
		FROM_BLOCK,
		CONCURRENCY,
		false,
		false,
		VERBOSE,
	)
	if err != nil {
		log.Panicln(err)
	}
}
