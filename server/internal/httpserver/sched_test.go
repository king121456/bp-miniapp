package httpserver

import (
	"context"
	"os"
	"testing"
	"time"

	"bp-server/internal/config"
	"bp-server/internal/model"
	"bp-server/internal/service"
	"bp-server/internal/wechat"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestReminderAndDelete(t *testing.T) {
	ctx := context.Background()
	db := testDB(t, "bp_test")
	key := os.Getenv("DATA_KEY")
	if len(key) != 64 {
		key = "00000000000000000000000000000000000000000000000000000000000000aa"
	}
	_ = db.Drop(ctx)
	app, err := service.New(config.Config{JWTSecret: "x0123456789abcdef", DataKey: key, WxAppID: "t", WxSecret: "t"}, db)
	if err != nil {
		t.Fatal(err)
	}
	app.Wx = wechat.Fake{}
	_ = app.EnsureIndexes(ctx)
	uid := bson.NewObjectID()
	_, _ = db.Collection("users").InsertOne(ctx, bson.M{"_id": uid, "openid": "t"})
	_, _ = db.Collection("meds").InsertOne(ctx, bson.M{"userId": uid, "name": app.Enc("药"), "times": []string{"08:00"}})
	_, _ = db.Collection("subscribe").InsertOne(ctx, bson.M{"userId": uid, "kind": "med", "remaining": 1})
	d1 := time.Date(2026, 10, 12, 8, 0, 0, 0, model.CST)
	app.Tick(ctx, d1)
	if n, _ := db.Collection("todos").CountDocuments(ctx, bson.M{"userId": uid}); n != 0 {
		t.Fatalf("day1 todos=%d want 0", n)
	}
	app.Tick(ctx, d1)
	app.Tick(ctx, d1.AddDate(0, 0, 1))
	if n, _ := db.Collection("todos").CountDocuments(ctx, bson.M{"userId": uid}); n != 1 {
		t.Fatalf("day2 todos=%d want 1", n)
	}
	for _, c := range []string{"meds", "subscribe", "todos"} {
		_, _ = db.Collection(c).DeleteMany(ctx, bson.M{"userId": uid})
	}
	_, _ = db.Collection("users").DeleteOne(ctx, bson.M{"_id": uid})
	_ = db.Drop(ctx)
}

func testDB(t *testing.T, name string) *mongo.Database {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	cli, err := mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(2 * time.Second))
	if err != nil || cli.Ping(ctx, nil) != nil {
		t.Skip("MongoDB not reachable, skipping")
	}
	return cli.Database(name)
}
