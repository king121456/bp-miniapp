package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bp-server/internal/config"
	"bp-server/internal/model"
	"bp-server/internal/service"
	"bp-server/internal/wechat"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type tc struct {
	t   *testing.T
	app *service.App
	srv http.Handler
}

func (c tc) do(method, path, tok string, body any) (int, map[string]any) {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	c.srv.ServeHTTP(rr, req)
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return rr.Code, out
}

func newTC(t *testing.T) (tc, context.Context) {
	ctx := context.Background()
	db := testDB(t, "bp_v12_test")
	_ = db.Drop(ctx)
	app, _ := service.New(config.Config{
		JWTSecret: "0123456789abcdef", DataKey: "00000000000000000000000000000000000000000000000000000000000000aa",
		MakeupDays: 7, VisitMaxDays: 365, VisitRemindTime: "19:00",
		WxAppID: "t", WxSecret: "t", MissWindow: 120 * time.Minute,
	}, db)
	app.Wx = wechat.Fake{}
	_ = app.EnsureIndexes(ctx)
	t.Cleanup(func() { _ = db.Drop(ctx) })
	return tc{t, app, New(app).Routes()}, ctx
}

func (c tc) user(ctx context.Context, openid string) (bson.ObjectID, string) {
	id := bson.NewObjectID()
	_, _ = c.app.C("users").InsertOne(ctx, bson.M{"_id": id, "openid": openid, "consent": bson.M{"version": "1"}})
	return id, c.app.Sign(id.Hex())
}

func TestMakeup(t *testing.T) {
	c, ctx := newTC(t)
	pid, pt := c.user(ctx, "p")
	fid, ft := c.user(ctx, "f")
	_, _ = c.app.C("family").InsertOne(ctx, bson.M{"patientId": pid, "memberId": fid})
	medID := bson.NewObjectIDFromTimestamp(time.Now().AddDate(0, 0, -10))
	_, _ = c.app.C("meds").InsertOne(ctx, bson.M{"_id": medID, "userId": pid, "name": c.app.Enc("药A"), "dose": c.app.Enc("1片"), "times": []string{"08:00"}})
	now := time.Now().In(model.CST)
	d := func(n int) string { return now.AddDate(0, 0, -n).Format("2006-01-02") }
	mk := "/api/meds/" + medID.Hex() + "/makeup"

	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(1), Time: "08:00"}); s != 200 {
		t.Fatalf("makeup yesterday: %d", s)
	}
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(1), Time: "08:00"}); s != 409 {
		t.Fatalf("duplicate makeup: %d", s)
	}
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(7), Time: "08:00"}); s != 400 {
		t.Fatalf("7 days ago should be outside window: %d", s)
	}
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(6), Time: "08:00"}); s != 200 {
		t.Fatalf("6 days ago should be allowed: %d", s)
	}
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(-1), Time: "08:00"}); s != 400 {
		t.Fatalf("future slot: %d", s)
	}
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(2), Time: "09:00"}); s != 400 {
		t.Fatalf("non-scheduled slot: %d", s)
	}
	if s, _ := c.do("POST", mk, ft, model.SlotIn{Date: d(2), Time: "08:00"}); s != 404 {
		t.Fatalf("family makeup must be rejected: %d", s)
	}
	_, h := c.do("GET", "/api/family/patients/"+pid.Hex()+"/meds/history?days=3", ft, nil)
	found := false
	for _, x := range h["data"].([]any) {
		it := x.(map[string]any)
		if it["date"] == d(1) {
			found = it["status"] == "taken" && it["isMakeup"] == true && it["recordedAt"] != nil
			if _, has := it["canMakeup"]; has {
				t.Fatal("family view must not have makeup buttons")
			}
		}
	}
	if !found {
		t.Fatal("family should see makeup tag")
	}
	if s, _ := c.do("DELETE", mk+"?date="+d(1)+"&time=08:00", pt, nil); s != 200 {
		t.Fatalf("revoke: %d", s)
	}
	_, cand := c.do("GET", "/api/meds/makeup", pt, nil)
	cnt := cand["data"].(map[string]any)["count"].(float64)
	if cnt < 4 {
		t.Fatalf("candidates count=%v", cnt)
	}
	_, _ = c.app.C("family_msgs").InsertOne(ctx, bson.M{"memberId": fid, "patientId": pid, "date": d(6), "text": "x", "createdAt": time.Now(),
		"items": bson.A{bson.M{"medId": medID, "date": d(6), "slot": "08:00"}}})
	_, fm := c.do("GET", "/api/family/messages", ft, nil)
	notes := fm["data"].([]any)[0].(map[string]any)["makeupNotes"].([]any)
	if len(notes) != 1 {
		t.Fatalf("family message should carry makeup note: %v", notes)
	}
	_, rp := c.do("GET", "/api/report?days=7", pt, nil)
	adh := rp["data"].(map[string]any)["adherence"].(map[string]any)
	if adh["makeup"].(float64) != 1 || adh["taken"].(float64) != 1 {
		t.Fatalf("report adherence %v", adh)
	}
}

func TestVisits(t *testing.T) {
	c, ctx := newTC(t)
	pid, pt := c.user(ctx, "p")
	fid, ft := c.user(ctx, "f")
	_, _ = c.app.C("family").InsertOne(ctx, bson.M{"patientId": pid, "memberId": fid})
	day := func(n int) string { return time.Now().In(model.CST).AddDate(0, 0, n).Format("2006-01-02") }

	s, r := c.do("POST", "/api/visits", pt, model.VisitIn{VisitDate: day(10), Hospital: "市一院", Department: "心内科", Note: "带病历"})
	if s != 200 || r["data"].(map[string]any)["needSubscribe"] == nil {
		t.Fatalf("create: %d %v", s, r)
	}
	v1 := r["data"].(map[string]any)["visit"].(map[string]any)["id"].(string)
	if s, _ := c.do("POST", "/api/visits", pt, model.VisitIn{VisitDate: day(20)}); s != 409 {
		t.Fatalf("second upcoming without replace: %d", s)
	}
	if s, _ := c.do("POST", "/api/visits", pt, model.VisitIn{VisitDate: day(-1)}); s != 400 {
		t.Fatalf("past date: %d", s)
	}
	if s, _ := c.do("POST", "/api/visits", pt, model.VisitIn{VisitDate: day(400)}); s != 400 {
		t.Fatalf(">1 year: %d", s)
	}
	s, r = c.do("POST", "/api/visits", pt, model.VisitIn{VisitDate: day(0), Hospital: "市一院", Replace: true})
	if s != 200 {
		t.Fatalf("replace: %d", s)
	}
	v2 := r["data"].(map[string]any)["visit"].(map[string]any)
	if v2["pushStatus"] != "skipped" || v2["todoId"] == nil {
		t.Fatalf("today visit should skip push and create todo: %v", v2)
	}
	_, l := c.do("GET", "/api/visits", pt, nil)
	hist := l["data"].(map[string]any)["history"].([]any)
	if len(hist) != 1 || hist[0].(map[string]any)["status"] != "cancelled" || hist[0].(map[string]any)["id"] != v1 {
		t.Fatalf("old visit should be cancelled: %v", hist)
	}
	_, fv := c.do("GET", "/api/family/patients/"+pid.Hex()+"/visit", ft, nil)
	fvd := fv["data"].(map[string]any)
	if _, has := fvd["note"]; has || fvd["hospital"] != "市一院" {
		t.Fatalf("family visit view: %v", fvd)
	}
	id := v2["id"].(string)
	for _, x := range []struct{ m, p string }{{"PUT", "/api/visits/" + id}, {"POST", "/api/visits/" + id + "/cancel"}, {"DELETE", "/api/visits/" + id}} {
		if s, _ := c.do(x.m, x.p, ft, model.VisitIn{VisitDate: day(3)}); s != 404 {
			t.Fatalf("family %s must be rejected: %d", x.m, s)
		}
	}
	_, _ = c.app.C("visits").DeleteMany(ctx, bson.M{})
	_, _ = c.app.C("todos").DeleteMany(ctx, bson.M{})
	_, _ = c.app.C("subscribe").InsertOne(ctx, bson.M{"userId": pid, "kind": "visit", "remaining": 1})
	vid := bson.NewObjectID()
	_, _ = c.app.C("visits").InsertOne(ctx, model.Visit{ID: vid, Patient: pid, VisitDate: day(1), RemindTime: "19:00", RemindAt: time.Now().Add(-time.Minute), Status: "upcoming", PushStatus: "pending"})
	c.app.VisitTick(ctx, time.Now())
	var got model.Visit
	_ = c.app.C("visits").FindOne(ctx, bson.M{"_id": vid}).Decode(&got)
	n, _ := c.app.C("todos").CountDocuments(ctx, bson.M{"kind": "visit", "refId": vid, "done": false})
	if got.PushStatus != "sent" || n != 1 {
		t.Fatalf("visit tick: push=%s todos=%d", got.PushStatus, n)
	}
	_, _ = c.app.C("visits").UpdateOne(ctx, bson.M{"_id": vid}, bson.M{"$set": bson.M{"visitDate": day(-1)}})
	_, _ = c.app.C("todos").UpdateMany(ctx, bson.M{"refId": vid}, bson.M{"$set": bson.M{"expiresAt": time.Now().Add(-time.Hour)}})
	c.app.ExpireVisits(ctx, time.Now())
	_ = c.app.C("visits").FindOne(ctx, bson.M{"_id": vid}).Decode(&got)
	n, _ = c.app.C("todos").CountDocuments(ctx, bson.M{"kind": "visit", "refId": vid, "done": false})
	if got.Status != "past" || n != 0 {
		t.Fatalf("expire: status=%s todos=%d", got.Status, n)
	}
}

func TestScheduleVersions(t *testing.T) {
	c, ctx := newTC(t)
	pid, pt := c.user(ctx, "p")
	now := time.Now().In(model.CST)
	d := func(n int) string { return now.AddDate(0, 0, -n).Format("2006-01-02") }
	medID := bson.NewObjectIDFromTimestamp(time.Now().AddDate(0, 0, -10))
	_, _ = c.app.C("meds").InsertOne(ctx, bson.M{"_id": medID, "userId": pid, "name": c.app.Enc("药B"), "dose": "", "times": []string{"09:00"},
		"schedules": bson.A{bson.M{"from": d(10), "times": []string{"08:00"}}, bson.M{"from": d(3), "times": []string{"09:00"}}}})
	mk := "/api/meds/" + medID.Hex() + "/makeup"
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(5), Time: "09:00"}); s != 400 {
		t.Fatalf("09:00 did not exist 5 days ago: %d", s)
	}
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(5), Time: "08:00"}); s != 200 {
		t.Fatalf("08:00 existed 5 days ago: %d", s)
	}
	if s, _ := c.do("POST", mk, pt, model.SlotIn{Date: d(2), Time: "08:00"}); s != 400 {
		t.Fatalf("08:00 no longer exists 2 days ago: %d", s)
	}
	_, h := c.do("GET", "/api/meds/history?days=7", pt, nil)
	for _, x := range h["data"].([]any) {
		it := x.(map[string]any)
		if (it["date"].(string) >= d(3) && it["time"] != "09:00") || (it["date"].(string) < d(3) && it["time"] != "08:00") {
			t.Fatalf("wrong slot in history: %v %v", it["date"], it["time"])
		}
	}
	_, rp := c.do("GET", "/api/report?days=7", pt, nil)
	if p := rp["data"].(map[string]any)["adherence"].(map[string]any)["planned"].(float64); p != 7 {
		t.Fatalf("planned=%v want 7", p)
	}
	if s, _ := c.do("PUT", "/api/meds/"+medID.Hex(), pt, map[string]any{"name": "药B", "times": []string{"07:00", "19:00"}}); s != 200 {
		t.Fatalf("update: %d", s)
	}
	var m model.Med
	_ = c.app.C("meds").FindOne(ctx, bson.M{"_id": medID}).Decode(&m)
	if len(m.Schedules) != 3 || m.Schedules[2].From != d(0) || len(m.TimesOn(d(1))) != 1 || m.TimesOn(d(1))[0] != "09:00" {
		t.Fatalf("schedules: %+v", m.Schedules)
	}
}

func TestSkip(t *testing.T) {
	c, ctx := newTC(t)
	pid, pt := c.user(ctx, "p")
	fid, ft := c.user(ctx, "f")
	_, _ = c.app.C("family").InsertOne(ctx, bson.M{"patientId": pid, "memberId": fid})
	now := time.Now().In(model.CST)
	d := func(n int) string { return now.AddDate(0, 0, -n).Format("2006-01-02") }
	medID := bson.NewObjectIDFromTimestamp(time.Now().AddDate(0, 0, -10))
	_, _ = c.app.C("meds").InsertOne(ctx, bson.M{"_id": medID, "userId": pid, "name": c.app.Enc("药C"), "dose": "", "times": []string{"08:00"}})
	base := "/api/meds/" + medID.Hex()
	if s, _ := c.do("POST", base+"/skip", pt, model.SlotIn{Date: d(-1), Time: "08:00"}); s != 400 {
		t.Fatalf("future skip: %d", s)
	}
	if s, _ := c.do("POST", base+"/skip", ft, model.SlotIn{Date: d(1), Time: "08:00"}); s != 404 {
		t.Fatalf("family skip must be rejected: %d", s)
	}
	if s, _ := c.do("POST", base+"/skip", pt, model.SlotIn{Date: d(1), Time: "08:00"}); s != 200 {
		t.Fatalf("skip: %d", s)
	}
	if s, _ := c.do("POST", base+"/skip", pt, model.SlotIn{Date: d(1), Time: "08:00"}); s != 409 {
		t.Fatalf("double skip: %d", s)
	}
	st := func(date string) map[string]any {
		_, h := c.do("GET", "/api/meds/history?days=3", pt, nil)
		for _, x := range h["data"].([]any) {
			if it := x.(map[string]any); it["date"] == date {
				return it
			}
		}
		return nil
	}
	if it := st(d(1)); it["status"] != "skipped" || it["canMakeup"] != true {
		t.Fatalf("history after skip: %v", it)
	}
	if s, _ := c.do("POST", base+"/makeup", pt, model.SlotIn{Date: d(1), Time: "08:00"}); s != 200 {
		t.Fatalf("makeup skipped: %d", s)
	}
	if it := st(d(1)); it["status"] != "taken" || it["isMakeup"] != true {
		t.Fatalf("after makeup: %v", it)
	}
	_, r := c.do("DELETE", base+"/makeup?date="+d(1)+"&time=08:00", pt, nil)
	if r["data"].(map[string]any)["status"] != "skipped" || st(d(1))["status"] != "skipped" {
		t.Fatalf("revoke should restore skipped: %v", r)
	}
	_, _ = c.app.C("skips").InsertOne(ctx, bson.M{"userId": pid, "medId": medID, "date": d(0), "slot": "08:00"})
	_, _ = c.app.C("checkins").DeleteMany(ctx, bson.M{"date": d(0)})
	c.app.FamilyDigest(ctx, d(0))
	var fm struct {
		Items []any `bson:"items"`
	}
	_ = c.app.C("family_msgs").FindOne(ctx, bson.M{"memberId": fid}).Decode(&fm)
	if len(fm.Items) != 0 {
		t.Fatalf("skipped slot must not alert family: %v", fm.Items)
	}
	if s, _ := c.do("DELETE", base+"/skip?date="+d(1)+"&time=08:00", pt, nil); s != 200 || st(d(1))["status"] != "missed" {
		t.Fatalf("unskip: %d", s)
	}
}

func TestOneUpcomingVisitIndex(t *testing.T) {
	c, ctx := newTC(t)
	pid := bson.NewObjectID()
	if _, err := c.app.C("visits").InsertOne(ctx, bson.M{"patientId": pid, "status": "upcoming"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.app.C("visits").InsertOne(ctx, bson.M{"patientId": pid, "status": "upcoming"}); err == nil {
		t.Fatal("unique index on upcoming visit missing")
	}
	if _, err := c.app.C("visits").InsertOne(ctx, bson.M{"patientId": pid, "status": "cancelled"}); err != nil {
		t.Fatal("cancelled visits must be allowed", err)
	}
}
