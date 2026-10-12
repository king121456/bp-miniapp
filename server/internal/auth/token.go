package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Tokens struct {
	Secret string
}

func New(secret string) *Tokens {
	return &Tokens{Secret: secret}
}

func (t *Tokens) Sign(uid string) string {
	body := base64.RawURLEncoding.EncodeToString([]byte(uid + "." + time.Now().Add(30*24*time.Hour).Format(time.RFC3339)))
	m := hmac.New(sha256.New, []byte(t.Secret))
	m.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (t *Tokens) Verify(tok string) (bson.ObjectID, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return bson.ObjectID{}, errors.New("bad token")
	}
	m := hmac.New(sha256.New, []byte(t.Secret))
	m.Write([]byte(parts[0]))
	sig, _ := base64.RawURLEncoding.DecodeString(parts[1])
	if !hmac.Equal(sig, m.Sum(nil)) {
		return bson.ObjectID{}, errors.New("bad sig")
	}
	raw, _ := base64.RawURLEncoding.DecodeString(parts[0])
	uid, exp, ok := strings.Cut(string(raw), ".")
	tm, err := time.Parse(time.RFC3339, exp)
	if !ok || err != nil || time.Now().After(tm) {
		return bson.ObjectID{}, errors.New("expired")
	}
	return bson.ObjectIDFromHex(uid)
}
