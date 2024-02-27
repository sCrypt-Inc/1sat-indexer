package main

import (
	"context"
	"encoding/json"
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
	"github.com/shruggr/1sat-indexer/ordlock"
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
	// log.Println("POSTGRES:", POSTGRES)
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

	err = ordinals.Initialize(db, rdb)
	if err != nil {
		log.Panic(err)
	}

	err = ordlock.Initialize(db, rdb)
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	err := indexer.Exec(
		true,
		true,
		handleTx,
		func(height uint32) error {
			return nil
		},
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

type Sale struct {
	Spend    lib.ByteString `json:"spend"`
	Outpoint *lib.Outpoint  `json:"outpoint"`
	Sale     bool           `json:"sale"`
	Tick     *string        `json:"tick,omitempty"`
	Id       *lib.Outpoint  `json:"id,omitempty"`
}

func handleTx(ctx *lib.IndexContext) error {
	ordinals.ParseInscriptions(ctx)

	sale := false
	if len(*ctx.Tx.Inputs[0].UnlockingScript) > 250 {
		sale = true
	}

	// if ctx.Height == nil || *ctx.Height == 0 {
	// 	return nil
	// }

	if _, ok := ctx.Txos[0].Data["bsv20"].(*ordinals.Bsv20); ok {
		rows, err := db.Query(context.Background(), `
			UPDATE bsv20_txos
			SET sale=$2, spend_height=$3, spend_idx=$4
			WHERE spend=$1
			RETURNING txid, vout, tick, id, sale`,
			ctx.Txid,
			sale,
			ctx.Height,
			ctx.Idx,
		)
		if err != nil {
			log.Panicln(err)
		}
		defer rows.Close()
		for rows.Next() {
			var txid []byte
			var vout uint32
			bsv20Sale := &Sale{
				Spend: ctx.Txid,
			}
			err := rows.Scan(&txid, &vout, &bsv20Sale.Tick, &bsv20Sale.Id, &bsv20Sale.Sale)
			if err != nil {
				log.Panicln(err)
			}
			bsv20Sale.Outpoint = lib.NewOutpoint(txid, vout)
			out, err := json.Marshal(bsv20Sale)
			if err != nil {
				log.Panicln(err)
			}
			log.Println("PUBLISHING BSV20 SALE", string(out))
			rdb.Publish(context.Background(), "bsv20sales", out)
		}
	} else {
		_, err := db.Exec(context.Background(), `
			UPDATE listings
			SET sale=$2, spend_height=$3, spend_idx=$4
			WHERE spend=$1`,
			ctx.Txid,
			sale,
			ctx.Height,
			ctx.Idx,
		)
		if err != nil {
			log.Panicln(err)
		}
	}
	return nil
}
