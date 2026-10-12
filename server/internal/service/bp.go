package service

import (
	"net/http"
	"time"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (a *App) ListBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	a.WriteBP(w, r, uid)
}

func (a *App) WriteBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	f := bson.M{"userId": uid}
	tr := bson.M{}
	if t, err := time.Parse(time.RFC3339, r.URL.Query().Get("from")); err == nil {
		tr["$gte"] = t
	}
	if t, err := time.Parse(time.RFC3339, r.URL.Query().Get("to")); err == nil {
		tr["$lte"] = t
	}
	if len(tr) > 0 {
		f["time"] = tr
	}
	cur, err := a.C("bp_records").Find(r.Context(), f, options.Find().SetSort(bson.M{"time": -1}).SetLimit(500))
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	var out []model.BP
	_ = cur.All(r.Context(), &out)
	for i := range out {
		FixBP(&out[i], a)
	}
	if out == nil {
		out = []model.BP{}
	}
	OK(w, out)
}

func (a *App) CreateBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in model.BPIn
	if ReadJSON(r, &in) != nil || !in.Valid() {
		Fail(w, 400, "invalid input")
		return
	}
	t := time.Now()
	if in.Time != nil {
		t = *in.Time
	}
	rec := model.BP{User: uid, Time: t, Sys: in.Sys, Dia: in.Dia, HR: in.HR, Tags: in.Tags, Note: a.Enc(in.Note)}
	res, err := a.C("bp_records").InsertOne(r.Context(), rec)
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	_, _ = a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "measure", "date": model.Today(), "done": false}, bson.M{"$set": bson.M{"done": true}})
	rec.ID = res.InsertedID.(bson.ObjectID)
	FixBP(&rec, a)
	OK(w, bson.M{"record": rec, "needSubscribe": a.NeedSubscribe(r, uid, "measure")})
}

func (a *App) UpdateBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	var in model.BPIn
	if err != nil || ReadJSON(r, &in) != nil || !in.Valid() {
		Fail(w, 400, "invalid input")
		return
	}
	set := bson.M{"sys": in.Sys, "dia": in.Dia, "hr": in.HR, "tags": in.Tags, "note": a.Enc(in.Note)}
	if in.Time != nil {
		set["time"] = *in.Time
	}
	res, err := a.C("bp_records").UpdateOne(r.Context(), bson.M{"_id": id, "userId": uid}, bson.M{"$set": set})
	if err != nil || res.MatchedCount == 0 {
		Fail(w, 404, "not found")
		return
	}
	OK(w, nil)
}

func (a *App) DeleteBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	res, err := a.C("bp_records").DeleteOne(r.Context(), bson.M{"_id": id, "userId": uid})
	if err != nil || res.DeletedCount == 0 {
		Fail(w, 404, "not found")
		return
	}
	OK(w, nil)
}

func (a *App) StatsBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	a.WriteStats(w, r, uid)
}

func (a *App) WriteStats(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	days := 7
	if r.URL.Query().Get("range") == "month" {
		days = 30
	}
	now := time.Now().In(model.CST)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, model.CST).AddDate(0, 0, -(days - 1))
	cur, err := a.C("bp_records").Find(r.Context(), bson.M{"userId": uid, "time": bson.M{"$gte": start}}, options.Find().SetSort(bson.M{"time": 1}))
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	var recs []model.BP
	_ = cur.All(r.Context(), &recs)
	type acc struct{ s, d, h, hn, n int }
	add := func(x *acc, b model.BP) {
		x.s += b.Sys
		x.d += b.Dia
		x.n++
		if b.HR > 0 {
			x.h += b.HR
			x.hn++
		}
	}
	fin := func(x acc) map[string]any {
		if x.n == 0 {
			return map[string]any{"sys": nil, "dia": nil, "hr": nil, "count": 0}
		}
		var hr any
		if x.hn > 0 {
			hr = model.Round1(float64(x.h) / float64(x.hn))
		}
		return map[string]any{"sys": model.Round1(float64(x.s) / float64(x.n)), "dia": model.Round1(float64(x.d) / float64(x.n)), "hr": hr, "count": x.n}
	}
	var all, morning, evening acc
	daily := map[string]*acc{}
	for _, b := range recs {
		t := b.Time.In(model.CST)
		add(&all, b)
		if t.Hour() < 12 {
			add(&morning, b)
		} else {
			add(&evening, b)
		}
		k := t.Format("2006-01-02")
		if daily[k] == nil {
			daily[k] = &acc{}
		}
		add(daily[k], b)
	}
	points := []map[string]any{}
	for i := 0; i < days; i++ {
		k := start.AddDate(0, 0, i).Format("2006-01-02")
		p := map[string]any{"date": k}
		if d := daily[k]; d != nil {
			for kk, v := range fin(*d) {
				p[kk] = v
			}
		} else {
			p["count"] = 0
		}
		points = append(points, p)
	}
	OK(w, bson.M{"range": days, "average": fin(all), "morning": fin(morning), "evening": fin(evening), "daily": points})
}
