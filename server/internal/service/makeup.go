package service

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func (a *App) History(ctx context.Context, uid bson.ObjectID, days int, includeNote bool) []map[string]any {
	now := time.Now().In(model.CST)
	from := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	mc, _ := a.C("meds").Find(ctx, bson.M{"userId": uid})
	var meds []model.MedFull
	_ = mc.All(ctx, &meds)
	cc, _ := a.C("checkins").Find(ctx, bson.M{"userId": uid, "date": bson.M{"$gte": from}})
	var cks []model.Checkin
	_ = cc.All(ctx, &cks)
	ck := map[string]model.Checkin{}
	for _, c := range cks {
		ck[c.Med.Hex()+c.Date+c.Slot] = c
	}
	sk := a.SkipSet(ctx, uid, from)
	out := []map[string]any{}
	for d := 0; d < days; d++ {
		date := now.AddDate(0, 0, -d).Format("2006-01-02")
		for _, m := range meds {
			for _, t := range m.TimesOn(date) {
				if !SlotExists(m, date, t) {
					continue
				}
				c, has := ck[m.ID.Hex()+date+t]
				st := SlotStatus(date, t, now, has)
				_, skipped := sk[m.ID.Hex()+date+t]
				if skipped && !has {
					st = "skipped"
				}
				it := map[string]any{"medId": m.ID, "name": a.Dec(m.Name), "dose": a.Dec(m.Dose), "date": date, "time": t,
					"scheduledAt": model.ScheduledAt(date, t), "status": st, "isMakeup": has && c.IsMakeup, "deleted": m.DeletedAt != nil}
				if has {
					it["checkedAt"] = c.At.In(model.CST)
					if c.IsMakeup && c.RecordedAt != nil {
						it["recordedAt"] = c.RecordedAt.In(model.CST)
					}
				}
				it["canMakeup"] = (st == "missed" || st == "skipped") && d < a.Cfg.MakeupDays
				it["canSkip"] = (st == "pending" || st == "missed") && d < a.Cfg.MakeupDays
				it["canUnskip"] = st == "skipped" && d < a.Cfg.MakeupDays
				it["canRevoke"] = has && c.IsMakeup && d < a.Cfg.MakeupDays
				out = append(out, it)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i]["scheduledAt"].(time.Time).After(out[j]["scheduledAt"].(time.Time))
	})
	return out
}

func (a *App) MedHistory(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 90 {
		days = 7
	}
	OK(w, a.History(r.Context(), uid, days, true))
}

func (a *App) FamilyMedHistory(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	pid, good := a.FamilyPatient(w, r, uid)
	if !good {
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 90 {
		days = 7
	}
	h := a.History(r.Context(), pid, days, false)
	for _, it := range h {
		delete(it, "canMakeup")
		delete(it, "canRevoke")
		delete(it, "canSkip")
		delete(it, "canUnskip")
	}
	OK(w, h)
}

func (a *App) MakeupCandidates(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	items := []map[string]any{}
	for _, it := range a.History(r.Context(), uid, a.Cfg.MakeupDays, true) {
		if it["canMakeup"] == true {
			items = append(items, it)
		}
	}
	OK(w, bson.M{"days": a.Cfg.MakeupDays, "count": len(items), "items": items})
}

func (a *App) makeupSlot(w http.ResponseWriter, r *http.Request, uid bson.ObjectID, in model.SlotIn) (model.MedFull, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	var m model.MedFull
	if err != nil || a.C("meds").FindOne(r.Context(), bson.M{"_id": id, "userId": uid}).Decode(&m) != nil {
		Fail(w, 404, "not found")
		return m, false
	}
	now := time.Now().In(model.CST)
	d, err := time.ParseInLocation("2006-01-02", in.Date, model.CST)
	today, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), model.CST)
	if err != nil || !hhmm.MatchString(in.Time) || !SlotExists(m, in.Date, in.Time) {
		Fail(w, 400, "no such scheduled slot")
		return m, false
	}
	if int(today.Sub(d).Hours()/24) >= a.Cfg.MakeupDays {
		Fail(w, 400, "outside makeup window")
		return m, false
	}
	return m, true
}

func (a *App) Makeup(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in model.SlotIn
	_ = ReadJSON(r, &in)
	m, good := a.makeupSlot(w, r, uid, in)
	if !good {
		return
	}
	now := time.Now()
	from := "missed"
	if a.IsSkipped(r.Context(), uid, m.ID, in.Date, in.Time) {
		from = "skipped"
	} else if st := SlotStatus(in.Date, in.Time, now.In(model.CST), false); st != "missed" {
		Fail(w, 400, "slot is "+st+", not missed or skipped")
		return
	}
	_, err := a.C("checkins").InsertOne(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time,
		"at": now, "isMakeup": true, "recordedAt": now, "makeupFrom": from, "scheduledAt": model.ScheduledAt(in.Date, in.Time)})
	if mongo.IsDuplicateKeyError(err) {
		Fail(w, 409, "already taken")
		return
	}
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	_, _ = a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "med", "refId": m.ID, "slot": in.Time, "date": in.Date},
		bson.M{"$set": bson.M{"done": true, "makeup": true}})
	_, _ = a.C("op_log").InsertOne(r.Context(), bson.M{"userId": uid, "op": "makeup", "medId": m.ID, "date": in.Date, "slot": in.Time, "at": now})
	OK(w, bson.M{"status": "taken", "isMakeup": true, "recordedAt": now.In(model.CST), "scheduledAt": model.ScheduledAt(in.Date, in.Time)})
}

func (a *App) RevokeMakeup(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	in := model.SlotIn{Date: r.URL.Query().Get("date"), Time: r.URL.Query().Get("time")}
	m, good := a.makeupSlot(w, r, uid, in)
	if !good {
		return
	}
	var c model.Checkin
	err := a.C("checkins").FindOneAndDelete(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time, "isMakeup": true}).Decode(&c)
	if err != nil {
		Fail(w, 404, "no makeup to revoke")
		return
	}
	_, _ = a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "med", "refId": m.ID, "slot": in.Time, "date": in.Date, "makeup": true},
		bson.M{"$set": bson.M{"makeup": false}})
	_, _ = a.C("op_log").InsertOne(r.Context(), bson.M{"userId": uid, "op": "revoke_makeup", "medId": m.ID, "date": in.Date, "slot": in.Time, "at": time.Now()})
	st := "missed"
	if c.MakeupFrom == "skipped" {
		st = "skipped"
	}
	OK(w, bson.M{"status": st})
}

func (a *App) FamilyMessages(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	cur, err := a.C("family_msgs").Find(r.Context(), bson.M{"memberId": uid}, nil)
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	var msgs []struct {
		ID      bson.ObjectID `bson:"_id"`
		Patient bson.ObjectID `bson:"patientId"`
		Text    string        `bson:"text"`
		Date    string        `bson:"date"`
		At      time.Time     `bson:"createdAt"`
		Items   []struct {
			Med  bson.ObjectID `bson:"medId"`
			Date string        `bson:"date"`
			Slot string        `bson:"slot"`
		} `bson:"items"`
	}
	_ = cur.All(r.Context(), &msgs)
	out := []map[string]any{}
	for i := len(msgs) - 1; i >= 0; i-- {
		mm := msgs[i]
		if n, _ := a.C("family").CountDocuments(r.Context(), bson.M{"patientId": mm.Patient, "memberId": uid}); n == 0 {
			continue
		}
		notes := []string{}
		for _, it := range mm.Items {
			var c model.Checkin
			if a.C("checkins").FindOne(r.Context(), bson.M{"medId": it.Med, "date": it.Date, "slot": it.Slot, "isMakeup": true}).Decode(&c) == nil && c.RecordedAt != nil {
				rt := c.RecordedAt.In(model.CST)
				notes = append(notes, it.Slot+" 的用药已于 "+strconv.Itoa(int(rt.Month()))+"月"+strconv.Itoa(rt.Day())+"日 "+rt.Format("15:04")+" 补记")
			}
		}
		out = append(out, map[string]any{"id": mm.ID, "patientId": mm.Patient, "date": mm.Date, "text": mm.Text, "createdAt": mm.At.In(model.CST), "makeupNotes": notes})
	}
	OK(w, out)
}

func (a *App) SkipSet(ctx context.Context, uid bson.ObjectID, from string) map[string]bool {
	cur, _ := a.C("skips").Find(ctx, bson.M{"userId": uid, "date": bson.M{"$gte": from}})
	var ss []struct {
		Med  bson.ObjectID `bson:"medId"`
		Date string        `bson:"date"`
		Slot string        `bson:"slot"`
	}
	_ = cur.All(ctx, &ss)
	out := map[string]bool{}
	for _, x := range ss {
		out[x.Med.Hex()+x.Date+x.Slot] = true
	}
	return out
}

func (a *App) IsSkipped(ctx context.Context, uid, med bson.ObjectID, date, slot string) bool {
	n, _ := a.C("skips").CountDocuments(ctx, bson.M{"userId": uid, "medId": med, "date": date, "slot": slot})
	return n > 0
}

func (a *App) Skip(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in model.SlotIn
	_ = ReadJSON(r, &in)
	if in.Date == "" {
		in.Date = model.Today()
	}
	m, good := a.makeupSlot(w, r, uid, in)
	if !good {
		return
	}
	st := SlotStatus(in.Date, in.Time, time.Now().In(model.CST), false)
	if st == "upcoming" {
		Fail(w, 400, "slot not yet due")
		return
	}
	if n, _ := a.C("checkins").CountDocuments(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time}); n > 0 {
		Fail(w, 409, "already taken")
		return
	}
	_, err := a.C("skips").InsertOne(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time,
		"at": time.Now(), "scheduledAt": model.ScheduledAt(in.Date, in.Time)})
	if mongo.IsDuplicateKeyError(err) {
		Fail(w, 409, "already skipped")
		return
	}
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	_, _ = a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "med", "refId": m.ID, "slot": in.Time, "date": in.Date},
		bson.M{"$set": bson.M{"done": true, "skipped": true}})
	_, _ = a.C("op_log").InsertOne(r.Context(), bson.M{"userId": uid, "op": "skip", "medId": m.ID, "date": in.Date, "slot": in.Time, "at": time.Now()})
	OK(w, bson.M{"status": "skipped"})
}

func (a *App) Unskip(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	in := model.SlotIn{Date: r.URL.Query().Get("date"), Time: r.URL.Query().Get("time")}
	m, good := a.makeupSlot(w, r, uid, in)
	if !good {
		return
	}
	if n, _ := a.C("checkins").CountDocuments(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time}); n > 0 {
		Fail(w, 409, "slot has a check-in; revoke the makeup first")
		return
	}
	res, err := a.C("skips").DeleteOne(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time})
	if err != nil || res.DeletedCount == 0 {
		Fail(w, 404, "not skipped")
		return
	}
	_, _ = a.C("op_log").InsertOne(r.Context(), bson.M{"userId": uid, "op": "unskip", "medId": m.ID, "date": in.Date, "slot": in.Time, "at": time.Now()})
	OK(w, bson.M{"status": SlotStatus(in.Date, in.Time, time.Now().In(model.CST), false)})
}
