package store

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store struct {
	DB *mongo.Database
}

func New(db *mongo.Database) *Store {
	return &Store{DB: db}
}

func (s *Store) C(name string) *mongo.Collection { return s.DB.Collection(name) }

func (s *Store) EnsureIndexes(ctx context.Context) error {
	idx := map[string][]mongo.IndexModel{
		"users":       {{Keys: bson.D{{Key: "openid", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"bp_records":  {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "time", Value: -1}}}},
		"meds":        {{Keys: bson.D{{Key: "userId", Value: 1}}}},
		"checkins":    {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "medId", Value: 1}, {Key: "date", Value: 1}, {Key: "slot", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"subscribe":   {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "kind", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"todos":       {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "done", Value: 1}, {Key: "createdAt", Value: -1}}}},
		"invites":     {{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}},
		"family":      {{Keys: bson.D{{Key: "patientId", Value: 1}, {Key: "memberId", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "memberId", Value: 1}}}},
		"visits":      {{Keys: bson.D{{Key: "patientId", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"status": "upcoming"}).SetName("one_upcoming_visit")}, {Keys: bson.D{{Key: "status", Value: 1}, {Key: "remindAt", Value: 1}}}},
		"family_msgs": {{Keys: bson.D{{Key: "memberId", Value: 1}, {Key: "createdAt", Value: -1}}}},
		"op_log":      {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "at", Value: -1}}}},
		"skips":       {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "medId", Value: 1}, {Key: "date", Value: 1}, {Key: "slot", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"sent_log":    {{Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(7 * 86400)}},
	}
	for c, m := range idx {
		if _, err := s.C(c).Indexes().CreateMany(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
