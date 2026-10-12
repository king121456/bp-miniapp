package httpserver

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNoResidue(t *testing.T) {
	ctx := context.Background()
	db := testDB(t, "bp")
	n, _ := db.Collection("users").CountDocuments(ctx, bson.M{"openid": "dev_p1"})
	if n != 0 {
		t.Fatal("user still exists")
	}
	for _, c := range []string{"bp_records", "meds", "checkins", "subscribe", "todos", "family", "invites"} {
		m, _ := db.Collection(c).CountDocuments(ctx, bson.M{})
		t.Logf("%s remaining docs: %d", c, m)
	}
}
