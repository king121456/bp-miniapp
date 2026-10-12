package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Config struct {
	Addr, MongoURI, DBName, JWTSecret, DataKey, WxAppID, WxSecret string
	TplMeasure, TplMed, TplFamily, TplVisit                       string
	MissWindow                                                    time.Duration
	MakeupDays, VisitMaxDays                                      int
	VisitRemindTime                                               string
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt(k string, d int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return d
}

func main() {
	cfg := Config{
		Addr: env("ADDR", ":8080"), MongoURI: env("MONGO_URI", "mongodb://localhost:27017"),
		DBName: env("MONGO_DB", "bp"), JWTSecret: env("JWT_SECRET", ""), DataKey: env("DATA_KEY", ""),
		WxAppID: env("WX_APPID", ""), WxSecret: env("WX_SECRET", ""),
		TplMeasure: env("WX_TPL_MEASURE", ""), TplMed: env("WX_TPL_MED", ""), TplFamily: env("WX_TPL_FAMILY", ""), TplVisit: env("WX_TPL_VISIT", ""),
		MissWindow: time.Duration(envInt("MISS_WINDOW_MINUTES", 120)) * time.Minute,
		MakeupDays: envInt("MAKEUP_WINDOW_DAYS", 7), VisitMaxDays: envInt("VISIT_MAX_DAYS", 365),
		VisitRemindTime: env("VISIT_REMIND_TIME", "19:00"),
	}
	if len(cfg.JWTSecret) < 16 || len(cfg.DataKey) != 64 {
		log.Fatal("JWT_SECRET (>=16 chars) and DATA_KEY (64 hex chars = AES-256 key) are required")
	}
	if os.Getenv("APP_ENV") != "dev" && (cfg.WxAppID == "" || cfg.WxSecret == "") {
		log.Fatal("WX_APPID and WX_SECRET are required unless APP_ENV=dev")
	}
	if cfg.WxAppID == "" {
		log.Print("WARNING: WX_APPID empty, fake login enabled (any code is accepted, openid=dev_<code>)")
	} else if os.Getenv("APP_ENV") == "dev" {
		log.Print("APP_ENV=dev: real WeChat login, dev CORS allowed")
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
	missWindow = cfg.MissWindow
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
