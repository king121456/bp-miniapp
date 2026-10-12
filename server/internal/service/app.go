package service

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"bp-server/internal/auth"
	"bp-server/internal/config"
	"bp-server/internal/crypto"
	"bp-server/internal/model"
	"bp-server/internal/store"
	"bp-server/internal/wechat"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// MissWindow is the leak-detection window; set from config at startup.
var MissWindow = 120 * time.Minute

type App struct {
	Cfg   config.Config
	Store *store.Store
	Crypt *crypto.Cipher
	Tokens *auth.Tokens
	Wx    wechat.WxAPI
}

func New(cfg config.Config, db *mongo.Database) (*App, error) {
	c, err := crypto.New(cfg.DataKey)
	if err != nil {
		return nil, err
	}
	if cfg.MissWindow > 0 {
		MissWindow = cfg.MissWindow
	}
	return &App{
		Cfg:   cfg,
		Store: store.New(db),
		Crypt: c,
		Tokens: auth.New(cfg.JWTSecret),
		Wx:    &wechat.Wx{AppID: cfg.WxAppID, Secret: cfg.WxSecret},
	}, nil
}

func (a *App) C(name string) *mongo.Collection { return a.Store.C(name) }
func (a *App) EnsureIndexes(ctx context.Context) error { return a.Store.EnsureIndexes(ctx) }
func (a *App) Enc(s string) string                    { return a.Crypt.Enc(s) }
func (a *App) Dec(s string) string                    { return a.Crypt.Dec(s) }
func (a *App) Sign(uid string) string                 { return a.Tokens.Sign(uid) }

func OK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": v})
}
func Fail(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": status, "message": msg})
}
func ReadJSON(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
}
func WriteJSON(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }

func SlotStatus(date, slot string, now time.Time, checked bool) string {
	if checked {
		return "taken"
	}
	s := model.ScheduledAt(date, slot)
	switch {
	case now.Before(s):
		return "upcoming"
	case now.Before(s.Add(MissWindow)):
		return "pending"
	}
	return "missed"
}

func SlotExists(m model.MedFull, date, slot string) bool {
	s := model.ScheduledAt(date, slot)
	if s.IsZero() || date < m.ID.Timestamp().In(model.CST).Format("2006-01-02") {
		return false
	}
	if m.DeletedAt != nil && !s.Before(*m.DeletedAt) {
		return false
	}
	for _, t := range m.TimesOn(date) {
		if t == slot {
			return true
		}
	}
	return false
}

func FixBP(b *model.BP, a *App) {
	b.Note = a.Dec(b.Note)
	b.Time = b.Time.In(model.CST)
	if b.Tags == nil {
		b.Tags = []string{}
	}
}

func (a *App) Tpl(kind string) string {
	return map[string]string{"measure": a.Cfg.TplMeasure, "med": a.Cfg.TplMed, "family": a.Cfg.TplFamily, "visit": a.Cfg.TplVisit}[kind]
}
