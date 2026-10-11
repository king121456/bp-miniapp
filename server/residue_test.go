package main

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestNoResidue(t *testing.T) {
	ctx := context.Background()
	cli, _ := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	db := cli.Database("bp")
	n, _ := db.Collection("users").CountDocuments(ctx, bson.M{"openid": "dev_p1"})
	if n != 0 {
		t.Fatal("user still exists")
	}
	for _, c := range []string{"bp_records", "meds", "checkins", "subscribe", "todos", "family", "invites"} {
		m, _ := db.Collection(c).CountDocuments(ctx, bson.M{})
		t.Logf("%s remaining docs: %d", c, m)
	}
}
