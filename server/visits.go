package main

import (
	"context"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Visit struct {
	ID         bson.ObjectID  `json:"id" bson:"_id,omitempty"`
	Patient    bson.ObjectID  `json:"-" bson:"patientId"`
	VisitDate  string         `json:"visitDate" bson:"visitDate"`
	Hospital   string         `json:"hospital" bson:"hospital"`
	Department string         `json:"department" bson:"department"`
	Note       string         `json:"note" bson:"note"`
	RemindTime string         `json:"remindTime" bson:"remindTime"`
	RemindAt   time.Time      `json:"remindAt" bson:"remindAt"`
	Status     string         `json:"status" bson:"status"`         // upcoming / past / cancelled
	PushStatus string         `json:"pushStatus" bson:"pushStatus"` // pending / sent / no_quota / failed / skipped
	TodoID     *bson.ObjectID `json:"todoId,omitempty" bson:"todoId,omitempty"`
	CreatedAt  time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt" bson:"updatedAt"`
}

type visitIn struct {
	VisitDate  string `json:"visitDate"`
	Hospital   string `json:"hospital"`
	Department string `json:"department"`
	Note       string `json:"note"`
	RemindTime string `json:"remindTime"` // 前一天几点提醒，默认 19:00
	Replace    bool   `json:"replace"`    // 已有即将到来的复诊时，确认替换
}

func (a *App) visitValid(w http.ResponseWriter, in *visitIn) bool {
	if in.RemindTime == "" {
		in.RemindTime = a.cfg.VisitRemindTime
	}
	now := time.Now().In(CST)
	d, err := time.ParseInLocation("2006-01-02", in.VisitDate, CST)
	today, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), CST)
	switch {
	case err != nil || d.Before(today) || d.After(today.AddDate(0, 0, a.cfg.VisitMaxDays)):
		fail(w, 400, "visitDate must be between today and "+strconv.Itoa(a.cfg.VisitMaxDays)+" days later")
	case utf8.RuneCountInString(in.Hospital) > 30 || utf8.RuneCountInString(in.Department) > 20 || utf8.RuneCountInString(in.Note) > 100:
		fail(w, 400, "hospital<=30, department<=20, note<=100 chars")
	case !hhmm.MatchString(in.RemindTime):
		fail(w, 400, "remindTime must be HH:MM")
	default:
		return true
	}
	return false
}

func visitRemindAt(date, hm string) time.Time {
	d, _ := time.ParseInLocation("2006-01-02", date, CST)
	return scheduledAt(d.AddDate(0, 0, -1).Format("2006-01-02"), hm)
}

func (a *App) outVisit(v Visit) Visit {
	v.Note = a.dec(v.Note)
	v.RemindAt = v.RemindAt.In(CST)
	return v
}

// GET /api/visits → { upcoming, history }
func (a *App) listVisits(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	a.expireVisits(r.Context(), time.Now().In(CST))
	cur, err := a.C("visits").Find(r.Context(), bson.M{"patientId": uid}, options.Find().SetSort(bson.M{"visitDate": -1}))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var vs []Visit
	cur.All(r.Context(), &vs)
	var up any
	hist := []Visit{}
	for _, v := range vs {
		if v.Status == "upcoming" {
			up = a.outVisit(v)
		} else {
			hist = append(hist, a.outVisit(v))
		}
	}
	ok(w, bson.M{"upcoming": up, "history": hist})
}

// POST /api/visits；已有即将到来的复诊时返回 409 和原复诊，带 replace:true 再提交则把原复诊记为已取消
func (a *App) createVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in visitIn
	if readJSON(r, &in) != nil {
		fail(w, 400, "invalid json")
		return
	}
	if !a.visitValid(w, &in) {
		return
	}
	ctx := r.Context()
	var old Visit
	if a.C("visits").FindOne(ctx, bson.M{"patientId": uid, "status": "upcoming"}).Decode(&old) == nil {
		if !in.Replace {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(409)
			writeJSON(w, map[string]any{"code": 409, "message": "upcoming visit exists", "data": a.outVisit(old)})
			return
		}
		a.cancel(ctx, old.ID)
	}
	now := time.Now()
	v := Visit{Patient: uid, VisitDate: in.VisitDate, Hospital: in.Hospital, Department: in.Department, Note: a.enc(in.Note),
		RemindTime: in.RemindTime, RemindAt: visitRemindAt(in.VisitDate, in.RemindTime), Status: "upcoming", PushStatus: "pending",
		CreatedAt: now, UpdatedAt: now}
	res, err := a.C("visits").InsertOne(ctx, v)
	if mongo.IsDuplicateKeyError(err) {
		fail(w, 409, "upcoming visit exists")
		return
	}
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	v.ID = res.InsertedID.(bson.ObjectID)
	v = a.scheduleVisit(ctx, v)
	ok(w, bson.M{"visit": a.outVisit(v), "needSubscribe": a.needSubscribe(r, uid, "visit")})
}

// PUT /api/visits/:id 修改后按新时间重新安排提醒，原提醒作废
func (a *App) updateVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	var in visitIn
	if err != nil || readJSON(r, &in) != nil {
		fail(w, 400, "invalid input")
		return
	}
	if !a.visitValid(w, &in) {
		return
	}
	ctx := r.Context()
	var v Visit
	if a.C("visits").FindOne(ctx, bson.M{"_id": id, "patientId": uid, "status": "upcoming"}).Decode(&v) != nil {
		fail(w, 404, "not found")
		return
	}
	a.dropVisitTodo(ctx, v)
	v.VisitDate, v.Hospital, v.Department, v.Note, v.RemindTime = in.VisitDate, in.Hospital, in.Department, a.enc(in.Note), in.RemindTime
	v.RemindAt, v.PushStatus, v.TodoID, v.UpdatedAt = visitRemindAt(in.VisitDate, in.RemindTime), "pending", nil, time.Now()
	a.C("visits").ReplaceOne(ctx, bson.M{"_id": id}, v)
	v = a.scheduleVisit(ctx, v)
	ok(w, bson.M{"visit": a.outVisit(v), "needSubscribe": a.needSubscribe(r, uid, "visit")})
}

func (a *App) cancelVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	var v Visit
	if a.C("visits").FindOne(r.Context(), bson.M{"_id": id, "patientId": uid, "status": "upcoming"}).Decode(&v) != nil {
		fail(w, 404, "not found")
		return
	}
	a.cancel(r.Context(), id)
	ok(w, nil)
}

func (a *App) cancel(ctx context.Context, id bson.ObjectID) {
	var v Visit
	if a.C("visits").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": "cancelled", "updatedAt": time.Now()}}).Decode(&v) == nil {
		a.dropVisitTodo(ctx, v)
	}
}

func (a *App) dropVisitTodo(ctx context.Context, v Visit) {
	a.C("todos").UpdateMany(ctx, bson.M{"userId": v.Patient, "kind": "visit", "refId": v.ID, "done": false}, bson.M{"$set": bson.M{"done": true, "expired": true}})
}

// DELETE /api/visits/:id 患者逐条删除（含历史）
func (a *App) deleteVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	var v Visit
	if a.C("visits").FindOneAndDelete(r.Context(), bson.M{"_id": id, "patientId": uid}).Decode(&v) != nil {
		fail(w, 404, "not found")
		return
	}
	a.dropVisitTodo(r.Context(), v)
	ok(w, nil)
}

// 家人只读：只看即将到来的复诊日期、医院、科室
func (a *App) familyVisit(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	pid, good := a.familyPatient(w, r, uid)
	if !good {
		return
	}
	var v Visit
	if a.C("visits").FindOne(r.Context(), bson.M{"patientId": pid, "status": "upcoming"}).Decode(&v) != nil {
		ok(w, nil)
		return
	}
	ok(w, bson.M{"visitDate": v.VisitDate, "hospital": v.Hospital, "department": v.Department})
}

// scheduleVisit：保存时如果提醒时间已过（含复诊就是今天），不推送，直接生成待办
func (a *App) scheduleVisit(ctx context.Context, v Visit) Visit {
	if time.Now().Before(v.RemindAt) {
		return v
	}
	return a.fireVisit(ctx, v, false)
}

func (a *App) fireVisit(ctx context.Context, v Visit, doPush bool) Visit {
	title := "复诊：" + mdLabel(v.VisitDate)
	if place := v.Hospital + v.Department; place != "" {
		title += " " + v.Hospital + " " + v.Department
	}
	extra := bson.M{"visitDate": v.VisitDate, "expiresAt": scheduledAt(v.VisitDate, "23:59").Add(time.Minute), "action": "report"}
	status := "skipped"
	if doPush {
		d, _ := time.ParseInLocation("2006-01-02", v.VisitDate, CST)
		msg := "明天（" + mdLabel(v.VisitDate) + "）复诊"
		if v.Hospital+v.Department != "" {
			msg += "：" + v.Hospital + v.Department
		}
		status = a.push(ctx, v.Patient, "visit", v.ID, v.RemindTime, d.AddDate(0, 0, -1).Format("2006-01-02"), title,
			map[string]string{"thing1": trunc(msg, 20), "time2": v.VisitDate}, true, extra)
		if status == "dup" {
			return v
		}
	} else {
		doc := bson.M{"userId": v.Patient, "kind": "visit", "refId": v.ID, "date": v.VisitDate, "title": title, "pushed": false, "done": false, "createdAt": time.Now()}
		for k, x := range extra {
			doc[k] = x
		}
		a.C("todos").InsertOne(ctx, doc)
	}
	var t struct {
		ID bson.ObjectID `bson:"_id"`
	}
	a.C("todos").FindOne(ctx, bson.M{"userId": v.Patient, "kind": "visit", "refId": v.ID, "done": false}).Decode(&t)
	v.PushStatus, v.TodoID = status, &t.ID
	a.C("visits").UpdateOne(ctx, bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"pushStatus": status, "todoId": t.ID}})
	return v
}

func mdLabel(date string) string {
	d, err := time.ParseInLocation("2006-01-02", date, CST)
	if err != nil {
		return date
	}
	return strconv.Itoa(int(d.Month())) + "月" + strconv.Itoa(d.Day()) + "日"
}

// visitTick：到点推送复诊提醒；复诊日过后改为已过、待办过期
func (a *App) visitTick(ctx context.Context, now time.Time) {
	cur, err := a.C("visits").Find(ctx, bson.M{"status": "upcoming", "pushStatus": "pending", "remindAt": bson.M{"$lte": now}})
	if err == nil {
		var vs []Visit
		cur.All(ctx, &vs)
		for _, v := range vs {
			a.fireVisit(ctx, v, true)
		}
	}
	a.expireVisits(ctx, now)
}

func (a *App) expireVisits(ctx context.Context, now time.Time) {
	today := now.In(CST).Format("2006-01-02")
	a.C("visits").UpdateMany(ctx, bson.M{"status": "upcoming", "visitDate": bson.M{"$lt": today}}, bson.M{"$set": bson.M{"status": "past", "updatedAt": time.Now()}})
	a.C("todos").UpdateMany(ctx, bson.M{"kind": "visit", "done": false, "expiresAt": bson.M{"$lte": now}}, bson.M{"$set": bson.M{"done": true, "expired": true}})
}
