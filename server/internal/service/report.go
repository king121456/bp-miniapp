package service

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (a *App) Report(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	q := r.URL.Query()
	now := time.Now().In(model.CST)
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, model.CST)
	var start time.Time
	if q.Get("from") != "" || q.Get("to") != "" {
		f, e1 := time.ParseInLocation("2006-01-02", q.Get("from"), model.CST)
		t, e2 := time.ParseInLocation("2006-01-02", q.Get("to"), model.CST)
		if e1 != nil || e2 != nil || t.Before(f) || t.Sub(f) > 89*24*time.Hour {
			Fail(w, 400, "from/to invalid or longer than 90 days")
			return
		}
		start, end = f, t
	} else {
		days, _ := strconv.Atoi(q.Get("days"))
		if days == 0 {
			days = 30
		}
		if days != 7 && days != 30 && days != 90 {
			Fail(w, 400, "days must be 7, 30 or 90")
			return
		}
		start = end.AddDate(0, 0, -(days - 1))
	}
	stop := end.AddDate(0, 0, 1)
	ndays := int(stop.Sub(start).Hours()/24 + 0.5)

	cur, err := a.C("bp_records").Find(r.Context(), bson.M{"userId": uid, "time": bson.M{"$gte": start, "$lt": stop}}, options.Find().SetSort(bson.M{"time": 1}))
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
	fin := func(x *acc) map[string]any {
		if x == nil || x.n == 0 {
			return nil
		}
		m := map[string]any{"sys": model.Round1(float64(x.s) / float64(x.n)), "dia": model.Round1(float64(x.d) / float64(x.n)), "hr": nil, "count": x.n}
		if x.hn > 0 {
			m["hr"] = model.Round1(float64(x.h) / float64(x.hn))
		}
		return m
	}
	var all, am, pm acc
	dayAM, dayPM := map[string]*acc{}, map[string]*acc{}
	symptoms := map[string]int{}
	for _, b := range recs {
		t := b.Time.In(model.CST)
		k := t.Format("2006-01-02")
		add(&all, b)
		tgt := dayPM
		if t.Hour() < 12 {
			add(&am, b)
			tgt = dayAM
		} else {
			add(&pm, b)
		}
		if tgt[k] == nil {
			tgt[k] = &acc{}
		}
		add(tgt[k], b)
		for _, s := range b.Tags {
			symptoms[s]++
		}
	}
	series := []map[string]any{}
	for i := 0; i < ndays; i++ {
		k := start.AddDate(0, 0, i).Format("2006-01-02")
		series = append(series, map[string]any{"date": k, "morning": fin(dayAM[k]), "evening": fin(dayPM[k])})
	}
	sym := []map[string]any{}
	for s, n := range symptoms {
		sym = append(sym, map[string]any{"name": s, "count": n})
	}
	sort.Slice(sym, func(i, j int) bool { return sym[i]["count"].(int) > sym[j]["count"].(int) })

	var full []model.MedFull
	mc, _ := a.C("meds").Find(r.Context(), bson.M{"userId": uid})
	_ = mc.All(r.Context(), &full)
	medOut := []map[string]any{}
	planned, taken, makeups := 0, 0, 0
	for _, m := range full {
		p := 0
		for i := 0; i < ndays; i++ {
			d := start.AddDate(0, 0, i).Format("2006-01-02")
			for _, t := range m.TimesOn(d) {
				if SlotExists(m, d, t) {
					p++
				}
			}
		}
		if p == 0 && m.DeletedAt != nil {
			continue
		}
		n, _ := a.C("checkins").CountDocuments(r.Context(), bson.M{"userId": uid, "medId": m.ID,
			"date": bson.M{"$gte": start.Format("2006-01-02"), "$lte": end.Format("2006-01-02")}})
		mk, _ := a.C("checkins").CountDocuments(r.Context(), bson.M{"userId": uid, "medId": m.ID, "isMakeup": true,
			"date": bson.M{"$gte": start.Format("2006-01-02"), "$lte": end.Format("2006-01-02")}})
		planned += p
		taken += int(n)
		makeups += int(mk)
		var rate any
		if p > 0 {
			rate = model.Round1(float64(n) * 100 / float64(p))
		}
		medOut = append(medOut, map[string]any{"name": a.Dec(m.Name), "dose": a.Dec(m.Dose), "times": m.Times, "planned": p, "taken": n, "rate": rate, "makeup": mk})
	}
	var totalRate any
	if planned > 0 {
		totalRate = model.Round1(float64(taken) * 100 / float64(planned))
	}
	OK(w, map[string]any{
		"from": start.Format("2006-01-02"), "to": end.Format("2006-01-02"), "days": ndays,
		"generatedAt": now,
		"average":     fin(&all), "morning": fin(&am), "evening": fin(&pm), "count": all.n,
		"series": series, "meds": medOut,
		"adherence": map[string]any{"planned": planned, "taken": taken, "makeup": makeups, "rate": totalRate},
		"symptoms":  sym,
		"footer":    "数据由用户自行记录，仅供就诊参考",
	})
}
