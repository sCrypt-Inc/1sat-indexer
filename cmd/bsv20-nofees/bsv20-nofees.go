package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/shruggr/1sat-indexer/indexer"
	"github.com/shruggr/1sat-indexer/lib"
	"github.com/shruggr/1sat-indexer/ordinals"
)

// var settled = make(chan uint32, 1000)
var POSTGRES string
var db *pgxpool.Pool
var rdb *redis.Client
var INDEXER string
var TOPIC string
var FROM_BLOCK uint
var VERBOSE int
var CONCURRENCY int
var ctx = context.Background()
var REVALIDATE bool

func init() {
	wd, _ := os.Getwd()
	log.Println("CWD:", wd)
	godotenv.Load(fmt.Sprintf(`%s/../../.env`, wd))

	flag.StringVar(&INDEXER, "id", "inscriptions", "Indexer name")
	flag.StringVar(&TOPIC, "t", "", "Junglebus SubscriptionID")
	flag.UintVar(&FROM_BLOCK, "s", uint(lib.TRIGGER), "Start from block")
	flag.IntVar(&CONCURRENCY, "c", 64, "Concurrency Limit")
	flag.IntVar(&VERBOSE, "v", 0, "Verbose")
	flag.BoolVar(&REVALIDATE, "r", false, "revalidate")
	flag.Parse()

	if POSTGRES == "" {
		POSTGRES = os.Getenv("POSTGRES_FULL")
	}
	log.Println("POSTGRES:", POSTGRES)

	if TOPIC == "" {
		TOPIC = os.Getenv("FULL_SUBSCRIPTIONID")
	}
	log.Println("TOPIC:", TOPIC)

	var err error

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

var limiter chan struct{}
var sub *redis.PubSub

func main() {
	limiter = make(chan struct{}, CONCURRENCY)
	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}

	subRdb := redis.NewClient(opts)
	sub = subRdb.Subscribe(ctx, "broadcast", "v2xfer")
	ch1 := sub.Channel()

	var wg sync.WaitGroup
	go func() {
		for msg := range ch1 {
			switch msg.Channel {
			case "broadcast":

				rawtx, err := base64.StdEncoding.DecodeString(msg.Payload)
				if err != nil {
					log.Println("[BROADCAST]: Decode Payload error")
					continue
				}
				limiter <- struct{}{} // will block if there is MAX structs in limiter
				wg.Add(1)

				go func(rawtx []byte) {
					defer func() {
						<-limiter // removes a struct from limiter, allowing another to proceed
						wg.Done()
						if r := recover(); r != nil {
							fmt.Println("Recovered in broadcast")
						}
					}()
					ctx, err := lib.ParseTxn(rawtx, "", 0, 0)
					if err != nil {
						log.Printf("[BROADCAST]: ParseTxn failed: %+v\n", err)
						return
					}
					ordinals.IndexInscriptions(ctx)
					ids := ordinals.IndexBsv20(ctx)

					for _, id := range ids {
						rdb.Publish(context.Background(), "v2xfer", fmt.Sprintf("%x:%s", ctx.Txid, id))
					}

					log.Printf("[BROADCAST]: succeed %x \n", ctx.Txid)
				}(rawtx)
			case "v2xfer":
				parts := strings.Split(msg.Payload, ":")
				txid, err := hex.DecodeString(parts[0])
				if err != nil {
					log.Println("Decode err", err)
					break
				}
				tokenId, err := lib.NewOutpointFromString(parts[1])
				if err != nil {
					log.Println("NewOutpointFromString err", err)
					break
				}

				nOutputs := ordinals.ValidateV2Transfer(txid, tokenId, false)
				log.Println("[V2XFER]: nOutputs=", msg.Payload, nOutputs)
			default:
			}
		}
	}()

	go func() {
		for {
			if !processV2() {
				log.Println("No work to do")
				time.Sleep(time.Second * 20)
			}
		}

	}()

	err = indexer.Exec(
		true,
		true,
		func(ctx *lib.IndexContext) error {
			ordinals.IndexInscriptions(ctx)
			ordinals.IndexBsv20(ctx)
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

func processV2() (didWork bool) {
	wg := sync.WaitGroup{}

	ids := ordinals.InitializeV2Ids()
	log.Println("Processing V2 ids len = ", len(ids))
	for _, outpoint := range ids {

		wg.Add(1)
		go func(outpoint *lib.Outpoint) {
			defer func() {
				limiter <- struct{}{}
				wg.Done()
			}()

			var sql string
			if REVALIDATE {
				sql = `
				SELECT txid, vout, height, idx, id, amt
				FROM bsv20_txos
				WHERE op='transfer' AND id=$1 AND status in (0, -1)
				ORDER BY height ASC, idx ASC, vout ASC
				LIMIT $2`
			} else {
				sql = `
				SELECT txid, vout, height, idx, id, amt
				FROM bsv20_txos
				WHERE op='transfer' AND id=$1 AND status = 0
				ORDER BY height ASC, idx ASC, vout ASC
				LIMIT $2`
			}

			rows, err := db.Query(ctx,
				sql,
				outpoint,
				2000,
			)
			if err != nil {
				log.Panic(err)
			}
			defer rows.Close()

			var prevTxid []byte
			for rows.Next() {
				bsv20 := &ordinals.Bsv20{}
				err = rows.Scan(&bsv20.Txid, &bsv20.Vout, &bsv20.Height, &bsv20.Idx, &bsv20.Id, &bsv20.Amt)
				if err != nil {
					log.Panicln(err)
				}
				//fmt.Printf("Validating Transfer: %s %x\n", outpoint.String(), bsv20.Txid)

				if bytes.Equal(prevTxid, bsv20.Txid) {
					// fmt.Printf("Skipping: %s %x\n", funds.Id.String(), bsv20.Txid)
					continue
				}
				prevTxid = bsv20.Txid
				ordinals.ValidateV2Transfer(bsv20.Txid, outpoint, bsv20.Height != nil)
				//fmt.Printf("Validated Transfer: %s %x\n", outpoint.String(), bsv20.Txid)
			}

		}(outpoint)

		<-limiter // will block if there is MAX structs in limiter
	}

	wg.Wait()
	log.Println("Processing V2 end ")
	return
}
