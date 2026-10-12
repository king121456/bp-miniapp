package service

import (
	"context"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (a *App) visitValid(w http.ResponseWriter, in *model.VisitIn) bool {
	if in.RemindTime == "" {
		in.RemindTime = a.Cfg.VisitRemindTime
	}
	now := time.Now().In(model.CST)
	d, err := time.ParseInLocation("2006-01-02", in.VisitDate, model.CST)
	today, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), model.CST)
	switch {
	case err != nil || d.Before(today) || d.After(today.AddDate(0, 0, a.Cfg.VisitMaxDays)):
		Fail(w, 400, "visitDate must be between today and "+strconv.Itoa(a.Cfg.VisitMaxDays)+" days later")
	case utf8.RuneCountInString(in.Hospital) > 30 || utf8.RuneCountInString(in.Department) > 20 || utf8.RuneCountInString(in.Note) > 100:
		Fail(w, 400, "hospital<=30, department<=20, note<=100 chars")
	case !hhmm.MatchString(in.RemindTime):
		Fail(w, 400, "remindTime must be HH:MM")
	default:
		return true
	}
	return false
}

func visitRemindAt(date, hm string) time.Time {
	d, _ := time.ParseInLocation("2006-01-02", date, model.CST)
	return model.ScheduledAt(d.AddDate(0, 0, -1).Format("2006-01-02"), hm)
}

func (a *App) OutVisit(v model.Visit) model.Visit {
	v.Note = a.Dec(v.Note)
	v.RemindAt = v.RemindAt.In(model.CST)
	return v
}

func (a *App) ListVisits(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	a.ExpireVisits(r.Context(), time.Now().In(model.CST))
	cur, err := a.C("visits").Find(r.Context(), bson.M{"patientId": uid}, options.Find().SetSort(bson.M{"visitDate": -1}))
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	var vs []model.Visit
	_ = cur.All(r.Context(), &vs)
	var up any
	hist := []model.Visit{}
	for _, v := range vs {
		if v.Status == "upcoming" {
			up = a.OutVisit(v)
		} else {
			hist = append(hist, a.OutVisit(v))
		}
	}
	OK(w, bson.M{"upcoming": up, "history": hist})
}

func (a *App) CreateVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in model.VisitIn
	if ReadJSON(r, &in) != nil {
		Fail(w, 400, "invalid json")
		return
	}
	if !a.visitValid(w, &in) {
		return
	}
	ctx := r.Context()
	var old model.Visit
	if a.C("visits").FindOne(ctx, bson.M{"patientId": uid, "status": "upcoming"}).Decode(&old) == nil {
		if !in.Replace {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(409)
			WriteJSON(w, map[string]any{"code": 409, "message": "upcoming visit exists", "data": a.OutVisit(old)})
			return
		}
		a.Cancel(ctx, old.ID)
	}
	now := time.Now()
	v := model.Visit{Patient: uid, VisitDate: in.VisitDate, Hospital: in.Hospital, Department: in.Department, Note: a.Enc(in.Note),
		RemindTime: in.RemindTime, RemindAt: visitRemindAt(in.VisitDate, in.RemindTime), Status: "upcoming", PushStatus: "pending",
		CreatedAt: now, UpdatedAt: now}
	res, err := a.C("visits").InsertOne(ctx, v)
	if mongo.IsDuplicateKeyError(err) {
		Fail(w, 409, "upcoming visit exists")
		return
	}
	if err != nil {
		Fail(w, 500, "db error")
		return
	}
	v.ID = res.InsertedID.(bson.ObjectID)
	v = a.ScheduleVisit(ctx, v)
	OK(w, bson.M{"visit": a.OutVisit(v), "needSubscribe": a.NeedSubscribe(r, uid, "visit")})
}

func (a *App) UpdateVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	var in model.VisitIn
	if err != nil || ReadJSON(r, &in) != nil {
		Fail(w, 400, "invalid input")
		return
	}
	if !a.visitValid(w, &in) {
		return
	}
	ctx := r.Context()
	var v model.Visit
	if a.C("visits").FindOne(ctx, bson.M{"_id": id, "patientId": uid, "status": "upcoming"}).Decode(&v) != nil {
		Fail(w, 404, "not found")
		return
	}
	a.DropVisitTodo(ctx, v)
	v.VisitDate, v.Hospital, v.Department, v.Note, v.RemindTime = in.VisitDate, in.Hospital, in.Department, a.Enc(in.Note), in.RemindTime
	v.RemindAt, v.PushStatus, v.TodoID, v.UpdatedAt = visitRemindAt(in.VisitDate, in.RemindTime), "pending", nil, time.Now()
	_, _ = a.C("visits").ReplaceOne(ctx, bson.M{"_id": id}, v)
	v = a.ScheduleVisit(ctx, v)
	OK(w, bson.M{"visit": a.OutVisit(v), "needSubscribe": a.NeedSubscribe(r, uid, "visit")})
}

func (a *App) CancelVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	var v model.Visit
	if a.C("visits").FindOne(r.Context(), bson.M{"_id": id, "patientId": uid, "status": "upcoming"}).Decode(&v) != nil {
		Fail(w, 404, "not found")
		return
	}
	a.Cancel(r.Context(), id)
	OK(w, nil)
}

func (a *App) Cancel(ctx context.Context, id bson.ObjectID) {
	var v model.Visit
	if a.C("visits").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": "cancelled", "updatedAt": time.Now()}}).Decode(&v) == nil {
		a.DropVisitTodo(ctx, v)
	}
}

func (a *App) DropVisitTodo(ctx context.Context, v model.Visit) {
	_, _ = a.C("todos").UpdateMany(ctx, bson.M{"userId": v.Patient, "kind": "visit", "refId": v.ID, "done": false}, bson.M{"$set": bson.M{"done": true, "expired": true}})
}

func (a *App) DeleteVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	var v model.Visit
	if a.C("visits").FindOneAndDelete(r.Context(), bson.M{"_id": id, "patientId": uid}).Decode(&v) != nil {
		Fail(w, 404, "not found")
		return
	}
	a.DropVisitTodo(r.Context(), v)
	OK(w, nil)
}

func (a *App) FamilyVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	pid, good := a.FamilyPatient(w, r, uid)
	if !good {
		return
	}
	var v model.Visit
	if a.C("visits").FindOne(r.Context(), bson.M{"patientId": pid, "status": "upcoming"}).Decode(&v) != nil {
		OK(w, nil)
		return
	}
	OK(w, bson.M{"visitDate": v.VisitDate, "hospital": v.Hospital, "department": v.Department})
}

func (a *App) ScheduleVisit(ctx context.Context, v model.Visit) model.Visit {
	if time.Now().Before(v.RemindAt) {
		return v
	}
	return a.FireVisit(ctx, v, false)
}

func (a *App) FireVisit(ctx context.Context, v model.Visit, doPush bool) model.Visit {
	title := "复诊：" + model.MDLabel(v.VisitDate)
	if place := v.Hospital + v.Department; place != "" {
		title += " " + v.Hospital + " " + v.Department
	}
	extra := bson.M{"visitDate": v.VisitDate, "expiresAt": model.ScheduledAt(v.VisitDate, "23:59").Add(time.Minute), "action": "report"}
	status := "skipped"
	if doPush {
		d, _ := time.ParseInLocation("2006-01-02", v.VisitDate, model.CST)
		msg := "明天（" + model.MDLabel(v.VisitDate) + "）复诊"
		if v.Hospital+v.Department != "" {
			msg += "：" + v.Hospital + v.Department
		}
		status = a.Push(ctx, v.Patient, "visit", v.ID, v.RemindTime, d.AddDate(0, 0, -1).Format("2006-01-02"), title,
			map[string]string{"thing1": model.Trunc(msg, 20), "time2": v.VisitDate}, true, extra)
		if status == "dup" {
			return v
		}
	} else {
		doc := bson.M{"userId": v.Patient, "kind": "visit", "refId": v.ID, "date": v.VisitDate, "title": title, "pushed": false, "done": false, "createdAt": time.Now()}
		for k, x := range extra {
			doc[k] = x
		}
		_, _ = a.C("todos").InsertOne(ctx, doc)
	}
	var t struct {
		ID bson.ObjectID `bson:"_id"`
	}
	_ = a.C("todos").FindOne(ctx, bson.M{"userId": v.Patient, "kind": "visit", "refId": v.ID, "done": false}).Decode(&t)
	v.PushStatus, v.TodoID = status, &t.ID
	_, _ = a.C("visits").UpdateOne(ctx, bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"pushStatus": status, "todoId": t.ID}})
	return v
}

func (a *App) VisitTick(ctx context.Context, now time.Time) {
	cur, err := a.C("visits").Find(ctx, bson.M{"status": "upcoming", "pushStatus": "pending", "remindAt": bson.M{"$lte": now}})
	if err == nil {
		var vs []model.Visit
		_ = cur.All(ctx, &vs)
		for _, v := range vs {
			a.FireVisit(ctx, v, true)
		}
	}
	a.ExpireVisits(ctx, now)
}

func (a *App) ExpireVisits(ctx context.Context, now time.Time) {
	today := now.In(model.CST).Format("2006-01-02")
	_, _ = a.C("visits").UpdateMany(ctx, bson.M{"status": "upcoming", "visitDate": bson.M{"$lt": today}}, bson.M{"$set": bson.M{"status": "past", "updatedAt": time.Now()}})
	_, _ = a.C("todos").UpdateMany(ctx, bson.M{"kind": "visit", "done": false, "expiresAt": bson.M{"$lte": now}}, bson.M{"$set": bson.M{"done": true, "expired": true}})
}
