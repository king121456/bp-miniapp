package service

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func code6() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return big.NewInt(0).Add(n, big.NewInt(100000)).String()
}

func (a *App) Invite(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	exp := time.Now().Add(24 * time.Hour)
	for i := 0; i < 5; i++ {
		c := code6()
		if _, err := a.C("invites").InsertOne(r.Context(), bson.M{"code": c, "patientId": uid, "expiresAt": exp}); err == nil {
			OK(w, bson.M{"code": c, "expiresAt": exp})
			return
		}
	}
	Fail(w, 500, "try again")
}

func (a *App) Bind(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in struct {
		Code string `json:"code"`
	}
	_ = ReadJSON(r, &in)
	var inv struct {
		Patient bson.ObjectID `bson:"patientId"`
	}
	err := a.C("invites").FindOneAndDelete(r.Context(), bson.M{"code": in.Code, "expiresAt": bson.M{"$gt": time.Now()}}).Decode(&inv)
	if err != nil {
		Fail(w, 404, "invalid or expired code")
		return
	}
	if inv.Patient == uid {
		Fail(w, 400, "cannot bind yourself")
		return
	}
	_, err = a.C("family").InsertOne(r.Context(), bson.M{"patientId": inv.Patient, "memberId": uid, "createdAt": time.Now()})
	if mongo.IsDuplicateKeyError(err) {
		Fail(w, 409, "already bound")
		return
	}
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	OK(w, bson.M{"patientId": inv.Patient})
}

func (a *App) ListFamily(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	load := func(f bson.M, other string) []bson.M {
		cur, _ := a.C("family").Find(r.Context(), f)
		var links []bson.M
		_ = cur.All(r.Context(), &links)
		out := []bson.M{}
		for _, l := range links {
			var u struct {
				Nickname string `bson:"nickname"`
			}
			_ = a.C("users").FindOne(r.Context(), bson.M{"_id": l[other]}).Decode(&u)
			out = append(out, bson.M{"id": l["_id"], "userId": l[other], other: l[other], "nickname": u.Nickname, "createdAt": l["createdAt"]})
		}
		return out
	}
	OK(w, bson.M{"members": load(bson.M{"patientId": uid}, "memberId"), "patients": load(bson.M{"memberId": uid}, "patientId")})
}

func (a *App) Unbind(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	res, _ := a.C("family").DeleteOne(r.Context(), bson.M{"_id": id, "$or": bson.A{bson.M{"patientId": uid}, bson.M{"memberId": uid}}})
	if res == nil || res.DeletedCount == 0 {
		Fail(w, 404, "not found")
		return
	}
	OK(w, nil)
}

func (a *App) FamilyPatient(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) (bson.ObjectID, bool) {
	pid, err := bson.ObjectIDFromHex(r.PathValue("pid"))
	if err == nil {
		if n, _ := a.C("family").CountDocuments(r.Context(), bson.M{"patientId": pid, "memberId": uid}); n > 0 {
			return pid, true
		}
	}
	Fail(w, 403, "forbidden")
	return pid, false
}

func (a *App) FamilyBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	if pid, good := a.FamilyPatient(w, r, uid); good {
		a.WriteBP(w, r, pid)
	}
}
func (a *App) FamilyStats(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	if pid, good := a.FamilyPatient(w, r, uid); good {
		a.WriteStats(w, r, pid)
	}
}
