package service

import (
	"net/http"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var kinds = map[string]bool{"measure": true, "med": true, "family": true, "visit": true}

func (a *App) GetMeasure(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var u struct {
		Measure model.Measure `bson:"measure"`
	}
	_ = a.C("users").FindOne(r.Context(), bson.M{"_id": uid}).Decode(&u)
	t := true
	if u.Measure.MorningEnabled == nil {
		u.Measure.MorningEnabled = &t
	}
	if u.Measure.EveningEnabled == nil {
		u.Measure.EveningEnabled = &t
	}
	OK(w, u.Measure)
}

func (a *App) PutMeasure(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var m model.Measure
	if ReadJSON(r, &m) != nil || (m.Morning != "" && !hhmm.MatchString(m.Morning)) || (m.Evening != "" && !hhmm.MatchString(m.Evening)) {
		Fail(w, 400, "invalid input")
		return
	}
	_, _ = a.C("users").UpdateOne(r.Context(), bson.M{"_id": uid}, bson.M{"$set": bson.M{"measure": m}})
	OK(w, m)
}

func (a *App) AddGrant(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in struct {
		Accepted []string `json:"accepted"`
	}
	if ReadJSON(r, &in) != nil {
		Fail(w, 400, "invalid input")
		return
	}
	for _, k := range in.Accepted {
		if !kinds[k] {
			continue
		}
		_, _ = a.C("subscribe").UpdateOne(r.Context(), bson.M{"userId": uid, "kind": k}, bson.M{"$inc": bson.M{"remaining": 1}}, options.UpdateOne().SetUpsert(true))
	}
	OK(w, a.Grants(r, uid))
}

func (a *App) Grants(r *http.Request, uid bson.ObjectID) map[string]int {
	cur, _ := a.C("subscribe").Find(r.Context(), bson.M{"userId": uid})
	var gs []struct {
		Kind string `bson:"kind"`
		N    int    `bson:"remaining"`
	}
	_ = cur.All(r.Context(), &gs)
	out := map[string]int{"measure": 0, "med": 0, "family": 0, "visit": 0}
	for _, g := range gs {
		out[g.Kind] = g.N
	}
	return out
}

func (a *App) NeedSubscribe(r *http.Request, uid bson.ObjectID, kind string) any {
	if a.Grants(r, uid)[kind] >= 2 {
		return nil
	}
	return bson.M{"kind": kind, "templateId": a.Tpl(kind)}
}

func (a *App) ListTodos(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	cur, err := a.C("todos").Find(r.Context(), bson.M{"userId": uid, "done": false}, options.Find().SetSort(bson.M{"createdAt": -1}).SetLimit(50).SetProjection(bson.M{"userId": 0}))
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

func (a *App) DoneTodo(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	_, _ = a.C("todos").UpdateOne(r.Context(), bson.M{"_id": id, "userId": uid}, bson.M{"$set": bson.M{"done": true}})
	OK(w, nil)
}
