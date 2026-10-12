package service

import (
	"context"
	"log/slog"
	"time"

	"bp-server/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const FamilyDigestAt = "21:30"

func (a *App) Tick(ctx context.Context, now time.Time) {
	hm, date := now.Format("15:04"), now.Format("2006-01-02")

	cur, err := a.C("users").Find(ctx, bson.M{"measure.enabled": true, "$or": bson.A{
		bson.M{"measure.morning": hm, "measure.morningEnabled": bson.M{"$ne": false}},
		bson.M{"measure.evening": hm, "measure.eveningEnabled": bson.M{"$ne": false}}}})
	if err == nil {
		var us []struct {
			ID bson.ObjectID `bson:"_id"`
		}
		_ = cur.All(ctx, &us)
		for _, u := range us {
			a.Remind(ctx, u.ID, "measure", bson.ObjectID{}, hm, date, "该测量血压了",
				map[string]string{"thing1": "测量血压", "time2": date + " " + hm})
		}
	}

	cur, err = a.C("meds").Find(ctx, bson.M{"times": hm, "deletedAt": nil})
	if err == nil {
		var ms []model.Med
		_ = cur.All(ctx, &ms)
		for _, m := range ms {
			if n, _ := a.C("checkins").CountDocuments(ctx, bson.M{"medId": m.ID, "date": date, "slot": hm}); n > 0 {
				continue
			}
			name := a.Dec(m.Name)
			a.Remind(ctx, m.User, "med", m.ID, hm, date, "该吃"+name+"了",
				map[string]string{"thing1": model.Trunc(name, 20), "thing2": model.Trunc(a.Dec(m.Dose), 20), "time3": date + " " + hm})
		}
	}

	a.VisitTick(ctx, now)

	if hm == FamilyDigestAt {
		a.FamilyDigest(ctx, date)
	}
}

func (a *App) Remind(ctx context.Context, uid bson.ObjectID, kind string, ref bson.ObjectID, slot, date, title string, data map[string]string) {
	a.Push(ctx, uid, kind, ref, slot, date, title, data, false, nil)
}

func (a *App) Push(ctx context.Context, uid bson.ObjectID, kind string, ref bson.ObjectID, slot, date, title string, data map[string]string, alwaysTodo bool, extra bson.M) string {
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
		if err := a.Wx.SendSubscribe(ctx, u.OpenID, a.Tpl(kind), "pages/index/index", data); err != nil {
			slog.Error("push subscribe failed", "key", key, "err", err)
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
	_, _ = a.C("todos").InsertOne(ctx, doc)
	return status
}

func (a *App) FamilyDigest(ctx context.Context, date string) {
	cur, err := a.C("family").Find(ctx, bson.M{})
	if err != nil {
		return
	}
	var links []struct {
		Patient bson.ObjectID `bson:"patientId"`
		Member  bson.ObjectID `bson:"memberId"`
	}
	_ = cur.All(ctx, &links)
	day, _ := time.ParseInLocation("2006-01-02", date, model.CST)
	for _, l := range links {
		missedMed := 0
		items := bson.A{}
		mc, _ := a.C("meds").Find(ctx, bson.M{"userId": l.Patient, "deletedAt": nil})
		var ms []model.Med
		_ = mc.All(ctx, &ms)
		for _, m := range ms {
			for _, t := range m.TimesOn(date) {
				if SlotStatus(date, t, time.Now(), false) != "missed" {
					continue
				}
				if a.IsSkipped(ctx, l.Patient, m.ID, date, t) {
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
		_ = a.C("users").FindOne(ctx, bson.M{"_id": l.Patient}, options.FindOne()).Decode(&p)
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
			msg += "有" + model.Itoa(missedMed) + "次用药未打卡"
		}
		_, _ = a.C("family_msgs").InsertOne(ctx, bson.M{"memberId": l.Member, "patientId": l.Patient, "date": date,
			"text": msg, "items": items, "createdAt": time.Now()})
		a.Remind(ctx, l.Member, "family", l.Patient, FamilyDigestAt, date, msg,
			map[string]string{"thing1": model.Trunc(msg, 20), "time2": date + " " + FamilyDigestAt})
	}
}
