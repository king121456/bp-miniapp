package main

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// missWindow 漏服判定窗口：计划时间过后这么久仍未打卡，记为漏服（MISS_WINDOW_MINUTES，默认 120）。
var missWindow = 120 * time.Minute

func scheduledAt(date, slot string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+slot, CST)
	return t
}

// slotStatus：upcoming 还没到点 / pending 待服（判定窗口内）/ missed 漏服 / taken 已服
func slotStatus(date, slot string, now time.Time, checked bool) string {
	if checked {
		return "taken"
	}
	s := scheduledAt(date, slot)
	switch {
	case now.Before(s):
		return "upcoming"
	case now.Before(s.Add(missWindow)):
		return "pending"
	}
	return "missed"
}

type Checkin struct {
	Med         bson.ObjectID `bson:"medId"`
	Date        string        `bson:"date"`
	Slot        string        `bson:"slot"`
	At          time.Time     `bson:"at"`
	IsMakeup    bool          `bson:"isMakeup"`
	RecordedAt  *time.Time    `bson:"recordedAt"`
	MakeupFrom  string        `bson:"makeupFrom"`
	ScheduledAt time.Time     `bson:"scheduledAt"`
}

type medFull struct {
	Med       `bson:",inline"`
	DeletedAt *time.Time `bson:"deletedAt"`
}

// slotExists：该药在这个时间点确实有计划（药品创建之后、删除之前，且是它的服药时间点）
func slotExists(m medFull, date, slot string) bool {
	s := scheduledAt(date, slot)
	if s.IsZero() || date < m.ID.Timestamp().In(CST).Format("2006-01-02") {
		return false
	}
	if m.DeletedAt != nil && !s.Before(*m.DeletedAt) {
		return false
	}
	for _, t := range m.timesOn(date) {
		if t == slot {
			return true
		}
	}
	return false
}

// history 生成最近 days 天（含今天）每个计划时段的状态
func (a *App) history(ctx context.Context, uid bson.ObjectID, days int, includeNote bool) []map[string]any {
	now := time.Now().In(CST)
	from := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	mc, _ := a.C("meds").Find(ctx, bson.M{"userId": uid})
	var meds []medFull
	mc.All(ctx, &meds)
	cc, _ := a.C("checkins").Find(ctx, bson.M{"userId": uid, "date": bson.M{"$gte": from}})
	var cks []Checkin
	cc.All(ctx, &cks)
	ck := map[string]Checkin{}
	for _, c := range cks {
		ck[c.Med.Hex()+c.Date+c.Slot] = c
	}
	out := []map[string]any{}
	for d := 0; d < days; d++ {
		date := now.AddDate(0, 0, -d).Format("2006-01-02")
		for _, m := range meds {
			for _, t := range m.timesOn(date) {
				if !slotExists(m, date, t) {
					continue
				}
				c, has := ck[m.ID.Hex()+date+t]
				st := slotStatus(date, t, now, has)
				it := map[string]any{"medId": m.ID, "name": a.dec(m.Name), "dose": a.dec(m.Dose), "date": date, "time": t,
					"scheduledAt": scheduledAt(date, t), "status": st, "isMakeup": has && c.IsMakeup, "deleted": m.DeletedAt != nil}
				if has {
					it["checkedAt"] = c.At.In(CST)
					if c.IsMakeup && c.RecordedAt != nil {
						it["recordedAt"] = c.RecordedAt.In(CST)
					}
				}
				it["canMakeup"] = st == "missed" && d < a.cfg.MakeupDays
				it["canRevoke"] = has && c.IsMakeup && d < a.cfg.MakeupDays
				out = append(out, it)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i]["scheduledAt"].(time.Time).After(out[j]["scheduledAt"].(time.Time))
	})
	return out
}

// GET /api/meds/history?days=7（最多 90）
func (a *App) medHistory(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 90 {
		days = 7
	}
	ok(w, a.history(r.Context(), uid, days, true))
}

// 家人只读：没有补记、撤销按钮
func (a *App) familyMedHistory(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	pid, good := a.familyPatient(w, r, uid)
	if !good {
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 90 {
		days = 7
	}
	h := a.history(r.Context(), pid, days, false)
	for _, it := range h {
		delete(it, "canMakeup")
		delete(it, "canRevoke")
	}
	ok(w, h)
}

// GET /api/meds/makeup → 最近 N 天可补记的时段，首页"最近 7 天有 N 次未打卡"用 count
func (a *App) makeupCandidates(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	items := []map[string]any{}
	for _, it := range a.history(r.Context(), uid, a.cfg.MakeupDays, true) {
		if it["canMakeup"] == true {
			items = append(items, it)
		}
	}
	ok(w, bson.M{"days": a.cfg.MakeupDays, "count": len(items), "items": items})
}

type slotIn struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

func (a *App) makeupSlot(w http.ResponseWriter, r *http.Request, uid bson.ObjectID, in slotIn) (medFull, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	var m medFull
	if err != nil || a.C("meds").FindOne(r.Context(), bson.M{"_id": id, "userId": uid}).Decode(&m) != nil {
		fail(w, 404, "not found")
		return m, false
	}
	now := time.Now().In(CST)
	d, err := time.ParseInLocation("2006-01-02", in.Date, CST)
	today, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), CST)
	if err != nil || !hhmm.MatchString(in.Time) || !slotExists(m, in.Date, in.Time) {
		fail(w, 400, "no such scheduled slot")
		return m, false
	}
	if int(today.Sub(d).Hours()/24) >= a.cfg.MakeupDays {
		fail(w, 400, "outside makeup window")
		return m, false
	}
	return m, true
}

// POST /api/meds/:id/makeup { date, time } 只补已漏服的时段
func (a *App) makeup(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in slotIn
	readJSON(r, &in)
	m, good := a.makeupSlot(w, r, uid, in)
	if !good {
		return
	}
	now := time.Now()
	if st := slotStatus(in.Date, in.Time, now.In(CST), false); st != "missed" {
		fail(w, 400, "slot is "+st+", not missed")
		return
	}
	_, err := a.C("checkins").InsertOne(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time,
		"at": now, "isMakeup": true, "recordedAt": now, "makeupFrom": "missed", "scheduledAt": scheduledAt(in.Date, in.Time)})
	if mongo.IsDuplicateKeyError(err) {
		fail(w, 409, "already taken")
		return
	}
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "med", "refId": m.ID, "slot": in.Time, "date": in.Date},
		bson.M{"$set": bson.M{"done": true, "makeup": true}})
	a.C("op_log").InsertOne(r.Context(), bson.M{"userId": uid, "op": "makeup", "medId": m.ID, "date": in.Date, "slot": in.Time, "at": now})
	ok(w, bson.M{"status": "taken", "isMakeup": true, "recordedAt": now.In(CST), "scheduledAt": scheduledAt(in.Date, in.Time)})
}

// DELETE /api/meds/:id/makeup?date=&time= 撤销补记，恢复为漏服
func (a *App) revokeMakeup(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	in := slotIn{Date: r.URL.Query().Get("date"), Time: r.URL.Query().Get("time")}
	m, good := a.makeupSlot(w, r, uid, in)
	if !good {
		return
	}
	res, err := a.C("checkins").DeleteOne(r.Context(), bson.M{"userId": uid, "medId": m.ID, "date": in.Date, "slot": in.Time, "isMakeup": true})
	if err != nil || res.DeletedCount == 0 {
		fail(w, 404, "no makeup to revoke")
		return
	}
	a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "med", "refId": m.ID, "slot": in.Time, "date": in.Date, "makeup": true},
		bson.M{"$set": bson.M{"makeup": false}})
	a.C("op_log").InsertOne(r.Context(), bson.M{"userId": uid, "op": "revoke_makeup", "medId": m.ID, "date": in.Date, "slot": in.Time, "at": time.Now()})
	ok(w, bson.M{"status": "missed"})
}

// GET /api/family/messages 家人的站内消息；涉及的时段若已补记，追加"已于……补记"
func (a *App) familyMessages(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	cur, err := a.C("family_msgs").Find(r.Context(), bson.M{"memberId": uid}, nil)
	if err != nil {
		fail(w, 500, "db error")
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
	cur.All(r.Context(), &msgs)
	out := []map[string]any{}
	for i := len(msgs) - 1; i >= 0; i-- {
		mm := msgs[i]
		// 解绑后不再显示该患者的消息
		if n, _ := a.C("family").CountDocuments(r.Context(), bson.M{"patientId": mm.Patient, "memberId": uid}); n == 0 {
			continue
		}
		notes := []string{}
		for _, it := range mm.Items {
			var c Checkin
			if a.C("checkins").FindOne(r.Context(), bson.M{"medId": it.Med, "date": it.Date, "slot": it.Slot, "isMakeup": true}).Decode(&c) == nil && c.RecordedAt != nil {
				rt := c.RecordedAt.In(CST)
				notes = append(notes, it.Slot+" 的用药已于 "+strconv.Itoa(int(rt.Month()))+"月"+strconv.Itoa(rt.Day())+"日 "+rt.Format("15:04")+" 补记")
			}
		}
		out = append(out, map[string]any{"id": mm.ID, "patientId": mm.Patient, "date": mm.Date, "text": mm.Text, "createdAt": mm.At.In(CST), "makeupNotes": notes})
	}
	ok(w, out)
}
