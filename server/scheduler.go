package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FamilyDigestAt is when family members get told about today's missed items.
const FamilyDigestAt = "21:30"

// RunScheduler ticks once a minute (run a single instance, or the sent_log dedup protects duplicates).
func (a *App) RunScheduler(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		now := time.Now().In(CST)
		a.tick(ctx, now)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (a *App) tick(ctx context.Context, now time.Time) {
	hm, date := now.Format("15:04"), now.Format("2006-01-02")

	// measurement reminders
	cur, err := a.C("users").Find(ctx, bson.M{"measure.enabled": true, "$or": bson.A{
		bson.M{"measure.morning": hm, "measure.morningEnabled": bson.M{"$ne": false}},
		bson.M{"measure.evening": hm, "measure.eveningEnabled": bson.M{"$ne": false}}}})
	if err == nil {
		var us []struct {
			ID bson.ObjectID `bson:"_id"`
		}
		cur.All(ctx, &us)
		for _, u := range us {
			a.remind(ctx, u.ID, "measure", bson.ObjectID{}, hm, date, "该测量血压了",
				map[string]string{"thing1": "测量血压", "time2": date + " " + hm})
		}
	}

	// medication reminders
	cur, err = a.C("meds").Find(ctx, bson.M{"times": hm, "deletedAt": nil})
	if err == nil {
		var ms []Med
		cur.All(ctx, &ms)
		for _, m := range ms {
			if n, _ := a.C("checkins").CountDocuments(ctx, bson.M{"medId": m.ID, "date": date, "slot": hm}); n > 0 {
				continue
			}
			name := a.dec(m.Name)
			a.remind(ctx, m.User, "med", m.ID, hm, date, "该吃"+name+"了",
				map[string]string{"thing1": trunc(name, 20), "thing2": trunc(a.dec(m.Dose), 20), "time3": date + " " + hm})
		}
	}

	a.visitTick(ctx, now)

	if hm == FamilyDigestAt {
		a.familyDigest(ctx, date)
	}
}

// remind pushes one subscribe message if the user has a grant left, otherwise (or on failure) writes an in-app todo.
func (a *App) remind(ctx context.Context, uid bson.ObjectID, kind string, ref bson.ObjectID, slot, date, title string, data map[string]string) {
	a.push(ctx, uid, kind, ref, slot, date, title, data, false, nil)
}

// push 发一次订阅消息，返回 sent / no_quota / failed / dup。
// alwaysTodo 为 true 时无论推送成败都写待办（复诊用）。
func (a *App) push(ctx context.Context, uid bson.ObjectID, kind string, ref bson.ObjectID, slot, date, title string, data map[string]string, alwaysTodo bool, extra bson.M) string {
	key := kind + ":" + uid.Hex() + ":" + ref.Hex() + ":" + date + ":" + slot
	if _, err := a.C("sent_log").InsertOne(ctx, bson.M{"key": key, "at": time.Now()}); mongo.IsDuplicateKeyError(err) {
		return "dup"
	}
	var u struct {
		OpenID string `bson:"openid"`
	}
	if a.C("users").FindOne(ctx, bson.M{"_id": uid}).Decode(&u) != nil {
		return "failed"
	}
	status := "no_quota"
	res := a.C("subscribe").FindOneAndUpdate(ctx, bson.M{"userId": uid, "kind": kind, "remaining": bson.M{"$gt": 0}}, bson.M{"$inc": bson.M{"remaining": -1}})
	if res.Err() == nil {
		if err := a.wx.SendSubscribe(ctx, u.OpenID, a.tpl(kind), "pages/index/index", data); err != nil {
			log.Printf("push %s failed: %v", key, err)
			status = "failed"
		} else {
			status = "sent"
		}
	}
	if status == "sent" && !alwaysTodo {
		return status
	}
	doc := bson.M{"userId": uid, "kind": kind, "refId": ref, "slot": slot, "date": date,
		"title": title, "pushed": status == "sent", "done": false, "createdAt": time.Now()}
	for k, v := range extra {
		doc[k] = v
	}
	a.C("todos").InsertOne(ctx, doc)
	return status
}

// familyDigest tells each bound member which items the patient has not recorded today. Facts only, no judgement.
func (a *App) familyDigest(ctx context.Context, date string) {
	cur, err := a.C("family").Find(ctx, bson.M{})
	if err != nil {
		return
	}
	var links []struct {
		Patient bson.ObjectID `bson:"patientId"`
		Member  bson.ObjectID `bson:"memberId"`
	}
	cur.All(ctx, &links)
	day, _ := time.ParseInLocation("2006-01-02", date, CST)
	for _, l := range links {
		missedMed := 0
		items := bson.A{}
		mc, _ := a.C("meds").Find(ctx, bson.M{"userId": l.Patient, "deletedAt": nil})
		var ms []Med
		mc.All(ctx, &ms)
		for _, m := range ms {
			for _, t := range m.timesOn(date) {
				if slotStatus(date, t, time.Now(), false) != "missed" {
					continue
				}
				// 用户主动标了"没吃"的不算漏服，不通知家人
				if a.isSkipped(ctx, l.Patient, m.ID, date, t) {
					continue
				}
				if n, _ := a.C("checkins").CountDocuments(ctx, bson.M{"medId": m.ID, "date": date, "slot": t}); n == 0 {
					missedMed++
					items = append(items, bson.M{"medId": m.ID, "date": date, "slot": t})
				}
			}
		}
		bpCount, _ := a.C("bp_records").CountDocuments(ctx, bson.M{"userId": l.Patient, "time": bson.M{"$gte": day, "$lt": day.AddDate(0, 0, 1)}})
		if missedMed == 0 && bpCount > 0 {
			continue
		}
		var p struct {
			Nickname string `bson:"nickname"`
		}
		a.C("users").FindOne(ctx, bson.M{"_id": l.Patient}, options.FindOne()).Decode(&p)
		who := p.Nickname
		if who == "" {
			who = "家人"
		}
		msg := who + "今天"
		if bpCount == 0 {
			msg += "还没记录血压"
		}
		if missedMed > 0 {
			if bpCount == 0 {
				msg += "，"
			}
			msg += "有" + itoa(missedMed) + "次用药未打卡"
		}
		a.C("family_msgs").InsertOne(ctx, bson.M{"memberId": l.Member, "patientId": l.Patient, "date": date,
			"text": msg, "items": items, "createdAt": time.Now()})
		a.remind(ctx, l.Member, "family", l.Patient, FamilyDigestAt, date, msg,
			map[string]string{"thing1": trunc(msg, 20), "time2": date + " " + FamilyDigestAt})
	}
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
func itoa(i int) string { return fmtInt(i) }
func fmtInt(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
