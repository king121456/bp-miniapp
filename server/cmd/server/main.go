package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"bp-server/internal/config"
	"bp-server/internal/httpserver"
	"bp-server/internal/scheduler"
	"bp-server/internal/service"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	config.LoadDotEnv(".env")
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cli, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	if err := cli.Ping(ctx, nil); err != nil {
		log.Fatal("mongo ping: ", err)
	}
	app, err := service.New(cfg, cli.Database(cfg.DBName))
	if err != nil {
		log.Fatal(err)
	}
	if err := app.EnsureIndexes(ctx); err != nil {
		log.Fatal(err)
	}
	go scheduler.Run(context.Background(), app)
	srv := httpserver.New(app)
	slog.Info("listening", "addr", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, srv.Routes()))
}
