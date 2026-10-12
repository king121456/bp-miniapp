package service

import (
	"net/http"
	"time"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (a *App) Export(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var bps []model.BP
	cur, _ := a.C("bp_records").Find(r.Context(), bson.M{"userId": uid}, options.Find().SetSort(bson.M{"time": 1}))
	_ = cur.All(r.Context(), &bps)
	bps = append([]model.BP{}, bps...)
	for i := range bps {
		FixBP(&bps[i], a)
	}
	var meds []model.Med
	cur, _ = a.C("meds").Find(r.Context(), bson.M{"userId": uid})
	_ = cur.All(r.Context(), &meds)
	for i := range meds {
		meds[i].Name, meds[i].Dose = a.Dec(meds[i].Name), a.Dec(meds[i].Dose)
	}
	meds = append([]model.Med{}, meds...)
	cks := []bson.M{}
	cur, _ = a.C("checkins").Find(r.Context(), bson.M{"userId": uid}, options.Find().SetProjection(bson.M{"_id": 0, "userId": 0}))
	_ = cur.All(r.Context(), &cks)
	w.Header().Set("Content-Disposition", `attachment; filename="bp-export.json"`)
	OK(w, bson.M{"exportedAt": time.Now(), "bloodPressure": bps, "medications": meds, "checkins": cks})
}

func (a *App) DeleteAccount(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	ctx := r.Context()
	for _, c := range []string{"bp_records", "meds", "checkins", "skips", "subscribe", "todos", "op_log"} {
		if _, err := a.C(c).DeleteMany(ctx, bson.M{"userId": uid}); err != nil {
			Fail(w, 500, "db error")
			return
		}
	}
	_, _ = a.C("invites").DeleteMany(ctx, bson.M{"patientId": uid})
	_, _ = a.C("visits").DeleteMany(ctx, bson.M{"patientId": uid})
	_, _ = a.C("family_msgs").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"patientId": uid}, bson.M{"memberId": uid}}})
	_, _ = a.C("family").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"patientId": uid}, bson.M{"memberId": uid}}})
	_, _ = a.C("users").DeleteOne(ctx, bson.M{"_id": uid})
	OK(w, nil)
}
