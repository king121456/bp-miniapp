package service

import (
	"net/http"
	"sort"
	"time"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (a *App) medIn(r *http.Request) (model.Med, bool) {
	var m model.Med
	if ReadJSON(r, &m) != nil || m.Name == "" || len(m.Name) > 100 || len(m.Dose) > 100 || len(m.Times) == 0 || len(m.Times) > 8 {
		return m, false
	}
	for _, t := range m.Times {
		if !hhmm.MatchString(t) {
			return m, false
		}
	}
	sort.Strings(m.Times)
	return m, true
}

func (a *App) ListMeds(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	cur, err := a.C("meds").Find(r.Context(), bson.M{"userId": uid, "deletedAt": nil})
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	var meds []model.Med
	_ = cur.All(r.Context(), &meds)
	date := model.Today()
	cc, _ := a.C("checkins").Find(r.Context(), bson.M{"userId": uid, "date": date})
	var cks []struct {
		Med  bson.ObjectID `bson:"medId"`
		Slot string        `bson:"slot"`
		At   time.Time     `bson:"at"`
	}
	_ = cc.All(r.Context(), &cks)
	done := map[string]time.Time{}
	for _, c := range cks {
		done[c.Med.Hex()+c.Slot] = c.At
	}
	out := []map[string]any{}
	for _, m := range meds {
		slots := []map[string]any{}
		all := true
		for _, t := range m.Times {
			at, d := done[m.ID.Hex()+t]
			s := map[string]any{"time": t, "done": d}
			if d {
				s["doneAt"] = at
			} else {
				all = false
			}
			slots = append(slots, s)
		}
		out = append(out, map[string]any{"id": m.ID, "name": a.Dec(m.Name), "dose": a.Dec(m.Dose), "times": m.Times, "date": date, "slots": slots, "done": all})
	}
	OK(w, out)
}

func (a *App) CreateMed(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	m, good := a.medIn(r)
	if !good {
		Fail(w, 400, "invalid input")
		return
	}
	name, dose := m.Name, m.Dose
	m.User, m.Name, m.Dose = uid, a.Enc(name), a.Enc(dose)
	m.Schedules = []model.Schedule{{From: model.Today(), Times: m.Times}}
	res, err := a.C("meds").InsertOne(r.Context(), m)
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	m.ID, m.Name, m.Dose = res.InsertedID.(bson.ObjectID), name, dose
	OK(w, m)
}

func (a *App) UpdateMed(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	m, good := a.medIn(r)
	if err != nil || !good {
		Fail(w, 400, "invalid input")
		return
	}
	var cur model.Med
	if a.C("meds").FindOne(r.Context(), bson.M{"_id": id, "userId": uid, "deletedAt": nil}).Decode(&cur) != nil {
		Fail(w, 404, "not found")
		return
	}
	scheds := cur.Schedules
	if len(scheds) == 0 {
		scheds = []model.Schedule{{From: cur.ID.Timestamp().In(model.CST).Format("2006-01-02"), Times: cur.Times}}
	}
	if !model.SameTimes(scheds[len(scheds)-1].Times, m.Times) {
		if scheds[len(scheds)-1].From == model.Today() {
			scheds[len(scheds)-1].Times = m.Times
		} else {
			scheds = append(scheds, model.Schedule{From: model.Today(), Times: m.Times})
		}
	}
	res, err := a.C("meds").UpdateOne(r.Context(), bson.M{"_id": id, "userId": uid, "deletedAt": nil},
		bson.M{"$set": bson.M{"name": a.Enc(m.Name), "dose": a.Enc(m.Dose), "times": m.Times, "schedules": scheds}})
	if err != nil || res.MatchedCount == 0 {
		Fail(w, 404, "not found")
		return
	}
	OK(w, nil)
}

func (a *App) DeleteMed(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	res, err := a.C("meds").UpdateOne(r.Context(), bson.M{"_id": id, "userId": uid, "deletedAt": nil}, bson.M{"$set": bson.M{"deletedAt": time.Now()}})
	if err != nil || res.MatchedCount == 0 {
		Fail(w, 404, "not found")
		return
	}
	OK(w, nil)
}

func (a *App) Checkin(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		Fail(w, 400, "bad id")
		return
	}
	var in struct {
		Time string `json:"time"`
	}
	_ = ReadJSON(r, &in)
	var m model.Med
	if a.C("meds").FindOne(r.Context(), bson.M{"_id": id, "userId": uid, "deletedAt": nil}).Decode(&m) != nil {
		Fail(w, 404, "not found")
		return
	}
	if in.Time == "" && len(m.Times) == 1 {
		in.Time = m.Times[0]
	}
	valid := false
	for _, t := range m.Times {
		if t == in.Time {
			valid = true
		}
	}
	if !valid {
		Fail(w, 400, "time must be one of the med's times")
		return
	}
	if SlotStatus(model.Today(), in.Time, time.Now().In(model.CST), false) == "missed" {
		var c bson.M
		if a.C("checkins").FindOne(r.Context(), bson.M{"userId": uid, "medId": id, "date": model.Today(), "slot": in.Time}).Decode(&c) != nil {
			Fail(w, 409, "slot missed, use makeup")
			return
		}
	}
	_, err = a.C("checkins").UpdateOne(r.Context(), bson.M{"userId": uid, "medId": id, "date": model.Today(), "slot": in.Time},
		bson.M{"$setOnInsert": bson.M{"at": time.Now(), "isMakeup": false, "scheduledAt": model.ScheduledAt(model.Today(), in.Time)}}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	_, _ = a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "med", "refId": id, "slot": in.Time, "date": model.Today()}, bson.M{"$set": bson.M{"done": true}})
	OK(w, bson.M{"needSubscribe": a.NeedSubscribe(r, uid, "med")})
}

func (a *App) ListCheckins(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	from := time.Now().In(model.CST).AddDate(0, 0, -30).Format("2006-01-02")
	cur, err := a.C("checkins").Find(r.Context(), bson.M{"userId": uid, "date": bson.M{"$gte": from}}, options.Find().SetSort(bson.M{"at": -1}).SetProjection(bson.M{"_id": 0, "userId": 0}))
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	var out []bson.M
	_ = cur.All(r.Context(), &out)
	if out == nil {
		out = []bson.M{}
	}
	OK(w, out)
}
