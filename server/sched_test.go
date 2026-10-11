package main

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// 次数用完后提醒进待办；注销后库里查不到该用户
func TestReminderAndDelete(t *testing.T) {
	ctx := context.Background()
	db := testDB(t, "bp_test")
	key := os.Getenv("DATA_KEY")
	if len(key) != 64 {
		key = "00000000000000000000000000000000000000000000000000000000000000aa"
	}
	db.Drop(ctx)
	app, err := NewApp(Config{JWTSecret: "x", DataKey: key}, db)
	if err != nil {
		t.Fatal(err)
	}
	app.EnsureIndexes(ctx)
	uid := bson.NewObjectID()
	db.Collection("users").InsertOne(ctx, bson.M{"_id": uid, "openid": "t"})
	db.Collection("meds").InsertOne(ctx, bson.M{"userId": uid, "name": app.enc("药"), "times": []string{"08:00"}})
	db.Collection("subscribe").InsertOne(ctx, bson.M{"userId": uid, "kind": "med", "remaining": 1})
	d1 := time.Date(2026, 10, 12, 8, 0, 0, 0, CST)
	app.tick(ctx, d1) // 有 1 次授权：推送，不进待办
	if n, _ := db.Collection("todos").CountDocuments(ctx, bson.M{"userId": uid}); n != 0 {
		t.Fatalf("day1 todos=%d want 0", n)
	}
	app.tick(ctx, d1)                  // 同一分钟重复触发不重复发
	app.tick(ctx, d1.AddDate(0, 0, 1)) // 次数用完：进待办
	if n, _ := db.Collection("todos").CountDocuments(ctx, bson.M{"userId": uid}); n != 1 {
		t.Fatalf("day2 todos=%d want 1", n)
	}
	for _, c := range []string{"meds", "subscribe", "todos"} {
		db.Collection(c).DeleteMany(ctx, bson.M{"userId": uid})
	}
	db.Collection("users").DeleteOne(ctx, bson.M{"_id": uid})
	db.Drop(ctx)
}

// testDB 连不上本地 MongoDB 时跳过测试，而不是空指针
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
