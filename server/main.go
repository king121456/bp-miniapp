package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Config struct {
	Addr, MongoURI, DBName, JWTSecret, DataKey, WxAppID, WxSecret string
	TplMeasure, TplMed, TplFamily                                 string
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	cfg := Config{
		Addr: env("ADDR", ":8080"), MongoURI: env("MONGO_URI", "mongodb://localhost:27017"),
		DBName: env("MONGO_DB", "bp"), JWTSecret: env("JWT_SECRET", ""), DataKey: env("DATA_KEY", ""),
		WxAppID: env("WX_APPID", ""), WxSecret: env("WX_SECRET", ""),
		TplMeasure: env("WX_TPL_MEASURE", ""), TplMed: env("WX_TPL_MED", ""), TplFamily: env("WX_TPL_FAMILY", ""),
	}
	if len(cfg.JWTSecret) < 16 || len(cfg.DataKey) != 64 {
		log.Fatal("JWT_SECRET (>=16 chars) and DATA_KEY (64 hex chars = AES-256 key) are required")
	}
	if os.Getenv("APP_ENV") != "dev" && (cfg.WxAppID == "" || cfg.WxSecret == "") {
		log.Fatal("WX_APPID and WX_SECRET are required unless APP_ENV=dev")
	}
	if os.Getenv("APP_ENV") == "dev" {
		log.Print("WARNING: dev mode, any login code is accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cli, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	if err := cli.Ping(ctx, nil); err != nil {
		log.Fatal("mongo ping: ", err)
	}
	app, err := NewApp(cfg, cli.Database(cfg.DBName))
	if err != nil {
		log.Fatal(err)
	}
	if err := app.EnsureIndexes(ctx); err != nil {
		log.Fatal(err)
	}
	go app.RunScheduler(context.Background())
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, app.Routes()))
}
