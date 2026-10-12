package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var CST = time.FixedZone("CST", 8*3600)

type App struct {
	cfg  Config
	db   *mongo.Database
	aead cipher.AEAD
	wx   *Wx
}

func NewApp(cfg Config, db *mongo.Database) (*App, error) {
	key, err := hex.DecodeString(cfg.DataKey)
	if err != nil {
		return nil, err
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	return &App{cfg: cfg, db: db, aead: aead, wx: &Wx{AppID: cfg.WxAppID, Secret: cfg.WxSecret}}, nil
}

func (a *App) C(name string) *mongo.Collection { return a.db.Collection(name) }

func (a *App) EnsureIndexes(ctx context.Context) error {
	idx := map[string][]mongo.IndexModel{
		"users":      {{Keys: bson.D{{Key: "openid", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"bp_records": {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "time", Value: -1}}}},
		"meds":       {{Keys: bson.D{{Key: "userId", Value: 1}}}},
		"checkins":   {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "medId", Value: 1}, {Key: "date", Value: 1}, {Key: "slot", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"subscribe":  {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "kind", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"todos":      {{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "done", Value: 1}, {Key: "createdAt", Value: -1}}}},
		"invites":    {{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}},
		"family":     {{Keys: bson.D{{Key: "patientId", Value: 1}, {Key: "memberId", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "memberId", Value: 1}}}},
		"sent_log":   {{Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(7 * 86400)}},
	}
	for c, m := range idx {
		if _, err := a.C(c).Indexes().CreateMany(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

// ---- field encryption (notes, med names) ----
func (a *App) enc(s string) string {
	if s == "" {
		return ""
	}
	n := make([]byte, a.aead.NonceSize())
	rand.Read(n)
	return base64.StdEncoding.EncodeToString(a.aead.Seal(n, n, []byte(s), nil))
}
func (a *App) dec(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) < a.aead.NonceSize() {
		return ""
	}
	p, err := a.aead.Open(nil, b[:a.aead.NonceSize()], b[a.aead.NonceSize():], nil)
	if err != nil {
		return ""
	}
	return string(p)
}

// ---- token: base64(userId.exp).hmac ----
func (a *App) sign(uid string) string {
	body := base64.RawURLEncoding.EncodeToString([]byte(uid + "." + time.Now().Add(30*24*time.Hour).Format(time.RFC3339)))
	m := hmac.New(sha256.New, []byte(a.cfg.JWTSecret))
	m.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
func (a *App) verify(tok string) (bson.ObjectID, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return bson.ObjectID{}, errors.New("bad token")
	}
	m := hmac.New(sha256.New, []byte(a.cfg.JWTSecret))
	m.Write([]byte(parts[0]))
	sig, _ := base64.RawURLEncoding.DecodeString(parts[1])
	if !hmac.Equal(sig, m.Sum(nil)) {
		return bson.ObjectID{}, errors.New("bad sig")
	}
	raw, _ := base64.RawURLEncoding.DecodeString(parts[0])
	uid, exp, ok := strings.Cut(string(raw), ".")
	t, err := time.Parse(time.RFC3339, exp)
	if !ok || err != nil || time.Now().After(t) {
		return bson.ObjectID{}, errors.New("expired")
	}
	return bson.ObjectIDFromHex(uid)
}

type ctxKey struct{}

func (a *App) auth(h func(http.ResponseWriter, *http.Request, bson.ObjectID)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := a.verify(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			fail(w, 401, "unauthorized")
			return
		}
		n, _ := a.C("users").CountDocuments(r.Context(), bson.M{"_id": uid})
		if n == 0 {
			fail(w, 401, "unauthorized")
			return
		}
		h(w, r, uid)
	}
}

func ok(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": v})
}
func fail(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"code": status, "message": msg})
}
func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
}

func writeJSON(w http.ResponseWriter, v any) { json.NewEncoder(w).Encode(v) }
