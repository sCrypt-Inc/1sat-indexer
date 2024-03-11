package main

import (
	"context"
	"encoding/base64"
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

	rdb := redis.NewClient(opts)

	err = indexer.Initialize(db, rdb)
	if err != nil {
		log.Panic(err)
	}

	err = ordinals.Initialize(indexer.Db, indexer.Rdb)
	if err != nil {
		log.Panic(err)
	}
}

var sub *redis.PubSub
var ctx = context.Background()

func main() {

	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}

	subRdb := redis.NewClient(opts)
	sub = subRdb.Subscribe(ctx, "broadcast")
	ch1 := sub.Channel()

	go func() {
		for msg := range ch1 {
			switch msg.Channel {
			case "broadcast":
				rawtx, err := base64.StdEncoding.DecodeString(msg.Payload)
				if err != nil {
					log.Println("[BROADCAST]: Decode Payload error")
					continue
				}

				go func(rawtx []byte) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Println("Recovered in broadcast")
						}
					}()
					ctx, err := lib.ParseTxn(rawtx, "", 0, 0)
					if err != nil {
						log.Printf("[BROADCAST]: ParseTxn failed: %+v\n", err)
						return
					}

					ctx.SaveSpends()

					handleTx(ctx)

					log.Printf("[BROADCAST]: succeed %x \n", ctx.Txid)
				}(rawtx)

			default:

			}
		}
	}()

	err = indexer.Exec(
		true,
		false,
		handleTx,
		handleBlock,
		INDEXER,
		TOPIC,
		FROM_BLOCK,
		CONCURRENCY,
		true,
		false,
		VERBOSE,
	)
	if err != nil {
		log.Panicln(err)
	}
}

func handleTx(tx *lib.IndexContext) error {
	ordinals.ParseInscriptions(tx)
	ordinals.CalculateOrigins(tx)

	tx.Save()

	xfers := map[string]*ordinals.Bsv20{}
	for _, txo := range tx.Txos {
		if bsv20, ok := txo.Data["bsv20"].(*ordinals.Bsv20); ok {
			bsv20.Save(txo)
			// if mined tx, skip and it will be validated by block handler
			if bsv20.Op == "transfer" {
				if bsv20.Ticker != "" {
					xfers[bsv20.Ticker] = bsv20
				} else {
					xfers[bsv20.Id.String()] = bsv20
				}
			}
		}
	}
	for _, bsv20 := range xfers {
		ordinals.ValidateV2Transfer(tx.Txid, bsv20.Id)
	}
	return nil
}

func handleBlock(height uint32) error {
	// only need for bsv20 v2
	// ordinals.ValidateBsv20Deploy(height - 6)
	// ordinals.ValidateBsv20Txos(height - 6)
	return nil
}
