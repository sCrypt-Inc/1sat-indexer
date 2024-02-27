package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GorillaPool/go-junglebus"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/shruggr/1sat-indexer/lib"
	"github.com/shruggr/1sat-indexer/ordinals"
)

var POSTGRES string
var CONCURRENCY int
var PORT int
var db *pgxpool.Pool
var rdb *redis.Client
var ctx = context.Background()
var jb *junglebus.Client

func init() {
	wd, _ := os.Getwd()
	log.Println("CWD:", wd)
	godotenv.Load(fmt.Sprintf(`%s/../../.env`, wd))

	if POSTGRES == "" {
		POSTGRES = os.Getenv("POSTGRES_FULL")
	}

	log.Println("POSTGRES:", POSTGRES)
	var err error
	config, err := pgxpool.ParseConfig(POSTGRES)
	if err != nil {
		log.Panic(err)
	}
	config.MaxConnIdleTime = 15 * time.Second

	db, err = pgxpool.NewWithConfig(context.Background(), config)

	// db, err = pgxpool.New(context.Background(), POSTGRES)
	if err != nil {
		log.Panic(err)
	}

	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(opts)

	JUNGLEBUS := os.Getenv("JUNGLEBUS")
	if JUNGLEBUS == "" {
		JUNGLEBUS = "https://junglebus.gorillapool.io"
	}

	jb, err = junglebus.New(
		junglebus.WithHTTP(JUNGLEBUS),
	)
	if err != nil {
		log.Panicln(err.Error())
	}

	ordinals.Initialize(db, rdb)
}

func main() {
	// flag.IntVar(&CONCURRENCY, "c", 64, "Concurrency Limit")
	flag.IntVar(&PORT, "p", 8082, "Port to listen on")

	app := fiber.New()
	app.Use(recover.New())
	app.Use(logger.New())

	app.Get("/yo", func(c *fiber.Ctx) error {
		return c.SendString("Yo!")
	})
	app.Get("/ord/:address", func(c *fiber.Ctx) error {
		address := c.Params("address")
		err := ordinals.RefreshAddress(ctx, address)
		if err != nil {
			log.Println("RefreshAddress", err)
			return err
		}
		c.SendStatus(http.StatusNoContent)
		return nil
	})

	app.Get("/origin/:origin/latest", func(c *fiber.Ctx) error {
		origin, err := lib.NewOutpointFromString(c.Params("origin"))
		if err != nil {
			log.Println("Parse origin", err)
			return err
		}

		outpoint, err := ordinals.GetLatestOutpoint(ctx, origin)
		if err != nil {
			log.Println("GetLatestOutpoint", err)
			return err
		}
		return c.Send(*outpoint)
	})

	app.Listen(fmt.Sprintf(":%d", PORT))
}
