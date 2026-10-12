package service

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (a *App) Auth(h func(http.ResponseWriter, *http.Request, bson.ObjectID)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := a.Tokens.Verify(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			Fail(w, 401, "unauthorized")
			return
		}
		n, _ := a.C("users").CountDocuments(r.Context(), bson.M{"_id": uid})
		if n == 0 {
			Fail(w, 401, "unauthorized")
			return
		}
		h(w, r, uid)
	}
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code     string         `json:"code"`
		Nickname string         `json:"nickname"`
		Consent  *model.Consent `json:"consent"`
	}
	if ReadJSON(r, &in) != nil {
		slog.Warn("login rejected", "reason", "invalid json")
		Fail(w, 400, "invalid json")
		return
	}
	if in.Code == "" {
		slog.Warn("login rejected", "reason", "code required")
		Fail(w, 400, "code required")
		return
	}
	openid, err := a.Wx.Code2Session(r.Context(), in.Code)
	if err != nil {
		// Full WeChat errcode/errmsg already logged in wechat package; log handler context here.
		slog.Error("login wechat failed", "err", err, "code_len", len(in.Code))
		Fail(w, 502, "wechat login failed")
		return
	}
	set := bson.M{"lastLoginAt": time.Now()}
	if in.Consent != nil && in.Consent.Version != "" {
		set["consent"] = in.Consent
	}
	if in.Nickname != "" {
		set["nickname"] = in.Nickname
	}
	var u struct {
		ID      bson.ObjectID  `bson:"_id"`
		Consent *model.Consent `bson:"consent"`
	}
	err = a.C("users").FindOneAndUpdate(r.Context(), bson.M{"openid": openid},
		bson.M{"$set": set, "$setOnInsert": bson.M{"createdAt": time.Now(),
			"measure": bson.M{"enabled": false, "morning": "07:30", "evening": "20:00"}}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&u)
	if err != nil {
		slog.Error("login db error", "err", err)
		Fail(w, 500, "db error")
		return
	}
	if u.Consent == nil {
		slog.Warn("login rejected", "reason", "consent required", "userId", u.ID.Hex())
		Fail(w, 403, "consent required")
		return
	}
	OK(w, bson.M{"token": a.Sign(u.ID.Hex()), "consent": u.Consent})
}

func (a *App) Me(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var u bson.M
	_ = a.C("users").FindOne(r.Context(), bson.M{"_id": uid}, options.FindOne().SetProjection(bson.M{"openid": 0})).Decode(&u)
	OK(w, u)
}
